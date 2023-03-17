package cache

import (
	"Rtp_Http_Flv/configure"
	"Rtp_Http_Flv/container/flv"
	"Rtp_Http_Flv/container/rtp"
	"Rtp_Http_Flv/protocol/hls"
	"Rtp_Http_Flv/protocol/httpflv"
	"Rtp_Http_Flv/protocol/quic"
	"Rtp_Http_Flv/utils"
	"fmt"
	"github.com/NOMADxzy/livego/av"
	"github.com/emirpasic/gods/lists/arraylist"
	"sync"
)

//type rtpQueueItem struct {
//	packet *RTPPacket
//	seq    uint16
//}

//[1,2,3,0,0]

type Queue struct {
	m  sync.RWMutex
	wg sync.WaitGroup
	//maxSize      int
	Ssrc              uint32 //队列所属的流
	ChannelKey        string
	FirstSeq          uint16          //第一个Rtp包的序号
	PaddingWindowSize int             //i+PW个包到了，第i包还没到，则对i执行重传
	queue             *arraylist.List //rtpPacket队列
	outChan           chan interface{}
	InChan            chan interface{}
	init              bool
	flvRecord         *FlvRecord      //解析flv结构
	FlvWriters        *arraylist.List //http-flv对象
	hlsWriter         *hls.Source
	flvFile           *utils.File //录制文件
	cache             *SegmentCache
	accPackets        int //记录收到包的数量
}

func NewQueue(ssrc uint32, key string, wz int, record *FlvRecord, flvFile *utils.File) *Queue {
	var hlsWriter *hls.Source
	if configure.ENABLE_HLS { //选择是否开启hls服务
		hlsWriter = hls.GetWriter(key)
	}
	return &Queue{
		queue:             arraylist.New(),
		PaddingWindowSize: wz,
		Ssrc:              ssrc,
		ChannelKey:        key,
		flvRecord:         record,
		flvFile:           flvFile,
		outChan:           make(chan interface{}, configure.RTP_QUEUE_CHAN_SIZE),
		InChan:            make(chan interface{}, configure.RTP_QUEUE_CHAN_SIZE),
		FlvWriters:        arraylist.New(),
		hlsWriter:         hlsWriter,
		cache:             NewCache(),
	}
}

func (q *Queue) RecvPacket() {
	for {
		p, ok := <-q.InChan
		if ok {
			q.Enqueue(p.(*rtp.RtpPack))
			if q.accPackets == q.PaddingWindowSize {
				q.Check()
			}
			for q.queue.Size() > q.PaddingWindowSize {
				protoRp := q.Dequeue()
				q.outChan <- protoRp
			}
		}
	}
}

//	func (q *queue) Play() {
//		for {
//			protoRp := q.Dequeue()
//			err := extractFlv(protoRp, q)
//			if err != nil {
//				q.flvRecord.Reset()
//			}
//			if q.queue.Size() <= q.PaddingWindowSize {
//				break
//			}
//		}
//	}
func (q *Queue) Play() {
	for {
		protoRp, ok := <-q.outChan
		if ok {
			err := q.extractFlv(protoRp)
			if err != nil {
				q.flvRecord.Reset()
			}
		} else {
			return
		}
	}
}

// 从rtp包中提取出flvTag，根据record信息组合分片，debug打印调试信息
func (rtpQueue *Queue) extractFlv(protoRp interface{}) error {
	record := rtpQueue.flvRecord

	if protoRp == nil { //该包丢失
		record.Reset()               //清空当前tag的缓存
		record.jumpToNextHead = true //跳到下一个tag头开始解析
		return nil
	}

	rp := protoRp.(*rtp.RtpPack)
	payload := rp.Payload
	marker := rp.Marker

	if record.jumpToNextHead {
		if !utils.IsTagHead(payload) {
			return nil
		} else {
			record.jumpToNextHead = false
		}
	}

	tmpBuf := make([]byte, 4)
	//fmt.Println("-----------------", rp.SequenceNumber, "-----------------")
	if int(rp.SequenceNumber)%100 == 0 {
		//rtpQueue.print()
	}

	if marker == byte(0) { //该帧未结束
		if record.flvTag == nil { //该帧是初始帧
			// Read tag size
			copy(tmpBuf[1:], payload[1:4])
			record.TagSize = int(uint32(tmpBuf[1])<<16 | uint32(tmpBuf[2])<<8 | uint32(tmpBuf[3]) + uint32(11))
			//fmt.Println("新建初始帧长度为", record.TagSize)
			record.flvTag = make([]byte, record.TagSize)

			copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
			record.pos += len(payload)
		} else { //该帧是中间帧
			copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
			record.pos += len(payload)
		}
	} else { //该帧是结束帧
		if record.flvTag == nil { //没有之前分片
			record.flvTag = payload
		} else { //有前面的分片
			//fmt.Println("pos===", pos)
			//fmt.Println(len(payload))
			copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
		}
		//得到一个flv tag

		//将flv数据发送到该流下的所有客户端
		//保存流的initialSegment发送到客户端才能播放
		if !rtpQueue.cache.full {
			p := &flv.Packet{}
			p.Parse(record.flvTag, false)

			rtpQueue.cache.Write(p)
		}
		for i := 0; i < rtpQueue.FlvWriters.Size(); i++ {
			val, f := rtpQueue.FlvWriters.Get(i)
			if f {
				writer := val.(*httpflv.FLVWriter)
				if writer.Closed {
					rtpQueue.FlvWriters.Remove(i)
				} else { //播放该分段
					if !writer.Init {
						err := rtpQueue.cache.SendInitialSegment(writer)
						if err != nil {
							return err
						}
						writer.Init = true
					}
					err := writer.Write(record.flvTag)
					utils.CheckError(err)
				}
			}
		}
		//发送到hlsServer中
		if rtpQueue.hlsWriter != nil {
			p := &av.Packet{}
			p.Parse(record.flvTag, true)
			err := rtpQueue.hlsWriter.Write(p)
			utils.CheckError(err)
		}

		//录制到文件中
		if rtpQueue.flvFile != nil {
			err := rtpQueue.flvFile.WriteTagDirect(record.flvTag)
			utils.CheckError(err)
		}
		//fmt.Println("rtp seq:", rp.SequenceNumber, ",payload size: ", len(flvTag), ",rtp timestamp: ", rp.Timestamp)

		record.Reset()

	}
	return nil
}

func (q *Queue) Enqueue(rp *rtp.RtpPack) {
	q.m.Lock()
	defer q.m.Unlock()

	if rp == nil {
		return
	}

	q.accPackets += 1

	seq := rp.SequenceNumber
	if q.queue.Size() == 0 { //队列中还没有元素
		q.FirstSeq = seq
		q.queue.Add(rp)
	} else {
		var relative int
		if q.FirstSeq > seq {
			if int(q.FirstSeq-seq) > 60000 { //序列号到头
				relative = 65536 - int(q.FirstSeq) + int(seq)
			} else { //过时的包
				fmt.Println("useless packet seq: ", seq, ", firstSeq: ", q.FirstSeq)
				return
			}
		} else {
			relative = int(seq - q.FirstSeq)
		}
		if relative <= q.queue.Size() { //没到队列终点
			q.queue.Set(relative, rp)
		} else {
			for i := q.queue.Size(); i <= relative; i++ {
				if i != relative {
					q.queue.Set(i, nil)
					continue
				}
				q.queue.Set(i, rp)
			}
		}

	}

}

func (q *Queue) runQuic(seq uint16) {
	fmt.Println("packet lost seq = ", seq, ", ssrc = ", q.Ssrc, "run quic request")
	pkt := quic.GetByQuic(q.Ssrc, seq)
	q.Enqueue(pkt)
}

func (q *Queue) Dequeue() interface{} {
	//检测要取的包是否存在，不在则重传
	rp, _ := q.queue.Get(0)
	if rp == nil {
		//重传
		q.runQuic(q.FirstSeq)
	}

	var protoRp interface{}
	protoRp, _ = q.queue.Get(0)
	q.m.Lock()
	q.queue.Remove(0)
	q.FirstSeq += 1
	q.m.Unlock()
	return protoRp
}

func (q *Queue) Check() int { //检查窗口内队列Rtp的存在
	re_trans := 0
	//rtpParser := parser.NewRtpParser()
	for i := 0; i < q.PaddingWindowSize; i++ {
		rp, _ := q.queue.Get(i)
		if rp == nil {
			q.runQuic(q.FirstSeq + uint16(i))
			re_trans += 1
		}
	}
	return re_trans
}
func (q *Queue) print() {
	fmt.Println("rtp队列长度：", q.queue.Size())
	fmt.Print("rtp队列：")
	for i := 0; i < q.queue.Size(); i++ {
		rp, _ := q.queue.Get(i)
		if rp == nil {
			fmt.Print(" nil")
		} else {
			fmt.Print(rp.(*rtp.RtpPack).SequenceNumber, " ")
		}
	}
	fmt.Println()

}

func (q *Queue) Close() {
	if q.flvFile != nil {
		q.flvFile.Close()
	}
	fmt.Println("stream closed ssrc = ", q.Ssrc)
}
