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
	"github.com/sirupsen/logrus"
	"runtime"
	"sync"
	"time"
)

//type rtpQueueItem struct {
//	packet *RTPPacket
//	seq    uint16
//}

//[1,2,3,0,0]

type Queue struct {
	m sync.RWMutex
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
	hlsWriter         *hls.Source     // hls服务
	flvFile           *utils.File     //录制文件
	cache             *SegmentCache
	accPackets        int    //记录收到包的数量
	accLoss           int    //记录丢失包的数量
	accFlvTags        int    // 记录收到的flvTag数量
	previousLostSeq   uint16 //三个连续的丢包说明发生了拥塞，去除队列前所有的nil，跳到下个有效包开始解析
	startTime         int64  //流开始传输的时间 unix毫秒
	delay             int
	App               *App
}

func NewQueue(ssrc uint32, key string, wz int, record *FlvRecord, flvFile *utils.File, app *App) *Queue {
	var hlsWriter *hls.Source
	if configure.Conf.ENABLE_HLS { //选择是否开启hls服务
		hlsWriter = hls.GetWriter(key)
	}
	return &Queue{
		queue:             arraylist.New(),
		PaddingWindowSize: wz,
		Ssrc:              ssrc,
		ChannelKey:        key,
		flvRecord:         record,
		flvFile:           flvFile,
		outChan:           make(chan interface{}, configure.Conf.RTP_QUEUE_CHAN_SIZE),
		InChan:            make(chan interface{}, configure.Conf.RTP_QUEUE_CHAN_SIZE),
		FlvWriters:        arraylist.New(),
		hlsWriter:         hlsWriter,
		cache:             NewCache(),
		App:               app,
	}
}

func (q *Queue) RecvPacket() {
	for {
		p, ok := <-q.InChan
		if ok {
			q.Enqueue(p.(*rtp.RtpPack))
			//fmt.Printf("队列长度%d\n", q.queue.Size())
			//if q.accPackets == q.PaddingWindowSize {
			//	q.Check()
			//}
			for q.queue.Size() > q.PaddingWindowSize { //重传区的必取，包括不存在的
				protoRp := q.Dequeue()
				_ = q.extractFlv(protoRp)
			}
			for {
				if q.isFirstOk() && q.queue.Size() > 1 { //等待区取到空包位置处为止,//最少保留一个包在队列中，否则入队列时无法计算相对位置
					protoRp := q.Dequeue()
					_ = q.extractFlv(protoRp)
				} else {
					break
				}
			}
		}
	}
}

func (q *Queue) PrintInfo() {
	for {
		_ = <-time.After(5 * time.Second)
		if q.Ssrc == 0 {
			return
		}

		lastSeq := uint16(0)
		if val, ok := q.queue.Get(q.queue.Size() - 1); ok {
			lastSeq = val.(*rtp.RtpPack).SequenceNumber
		} else {
			if q.queue.Size() > 0 {
				panic("rtpQueue params error")
			}
		}
		configure.Log.WithFields(logrus.Fields{
			"length":     q.queue.Size(),
			"FirstSeq":   q.FirstSeq,
			"LastSeq":    lastSeq,
			"accRtpRecv": q.accPackets,
			"accFlvRecv": q.accFlvTags,
		}).Debugf("[ssrc=%d]current rtpQueue",
			q.Ssrc)
	}
}

func (q *Queue) Play() {
	for {
		protoRp, ok := <-q.outChan
		if ok {
			_ = q.extractFlv(protoRp)
		} else {
			return
		}
	}
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
		if utils.FirstBeforeSecond(seq, q.FirstSeq) {
			configure.Log.Errorf("[%v]useless packet seq: %v, firstSeq: %v", q.Ssrc, seq, q.FirstSeq)
			return
		} else {
			if seq > q.FirstSeq {
				relative = int(seq - q.FirstSeq)
			} else {
				relative = int(uint16(65535) - q.FirstSeq + seq + uint16(1))
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
}

func (q *Queue) runQuic(seq uint16) {
	q.accLoss += 1

	if seq == q.previousLostSeq+uint16(1) && configure.Conf.PROTECT { //出现连续丢包
		nils := q.getHeadNil()

		if nils > 5 {
			q.reshape() //删除开头的所有nil，防止堵塞
			q.App.UdpBufferSize *= 2
			if q.App.UdpBufferSize > configure.MAX_UDP_CACHE_SIZE {
				q.App.UdpBufferSize /= 2
			} else {
				err := q.App.UdpConn.SetReadBuffer(q.App.UdpBufferSize)
				if err != nil {
					defer func() {
						if x := recover(); x != nil {
							configure.Log.Errorf("runtime error: %v", x)
						}
					}()
					panic(fmt.Sprintf("set read buffer error: %v", err))
				}
			}
			q.flvRecord.Reset()
			q.flvRecord.jumpToNextHead = true
			configure.Log.Errorf("[warning] Continuous packet loss, reshaping queue, %d packets removed, change udp buffer size to %vKB", nils, q.App.UdpBufferSize/1024)
		}

	}
	pkt := quic.GetByQuic(q.Ssrc, seq)

	q.previousLostSeq = seq
	q.Enqueue(pkt)
}

func (q *Queue) Dequeue() interface{} {
	//检测要取的包是否存在，不在则重传
	if !q.isFirstOk() {
		//重传
		q.runQuic(q.FirstSeq)
	}

	protoRp, _ := q.queue.Get(0)
	q.m.Lock()
	q.queue.Remove(0)
	q.FirstSeq += 1
	q.m.Unlock()
	return protoRp
}

// 从rtp包中提取出flvTag，根据record信息组合分片，debug打印调试信息
func (q *Queue) extractFlv(protoRp interface{}) error {
	record := q.flvRecord

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
			if record.pos+len(payload) > record.TagSize { //越界
				record.Reset()               //清空当前tag的缓存
				record.jumpToNextHead = true //跳到下一个tag头开始解析
				return nil
			}
			copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
			record.pos += len(payload)
		}
	} else { //该帧是结束帧，获得一个flvTag
		if record.flvTag == nil { //没有之前分片
			record.flvTag = payload
		} else { //有前面的分片
			//fmt.Println("pos===", pos)
			//fmt.Println(len(payload))
			if record.pos+len(payload) > record.TagSize { //越界
				record.Reset()               //清空当前tag的缓存
				record.jumpToNextHead = true //跳到下一个tag头开始解析
				return nil
			}
			copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
		}
		q.accFlvTags += 1
		//得到一个flv tag,计算时延
		if q.accFlvTags%400 == 0 {
			q.getDelay()
		}

		//将flv数据发送到该流下的所有客户端
		//保存流的initialSegment发送到客户端才能播放
		if !q.cache.full {
			p := &flv.Packet{}
			p.Parse(record.flvTag, false)

			q.cache.Write(p)
		}
		for i := q.FlvWriters.Size() - 1; i >= 0; i-- {
			val, f := q.FlvWriters.Get(i)
			if f {
				if val == nil {
					q.FlvWriters.Remove(i)
					runtime.GC()
					continue
				}
				writer := val.(*httpflv.FLVWriter)
				if writer.Closed {
					writer.Close()
					q.FlvWriters.Remove(i)
					runtime.GC()
				} else { //播放该分段
					if !writer.Init {
						err := q.cache.SendInitialSegment(writer)
						if err != nil {
							q.flvRecord.Reset()
							return nil
						}
						writer.Init = true
					}
					err := writer.Write(record.flvTag)
					if err != nil {
						q.FlvWriters.Remove(i)
						runtime.GC()
					}
				}
			}
		}
		//发送到hlsServer中
		if q.hlsWriter != nil {
			p := &av.Packet{}
			p.Parse(record.flvTag, true)
			err := q.hlsWriter.Write(p)
			utils.CheckError(err)
		}

		//录制到文件中
		if q.flvFile != nil {
			err := q.flvFile.WriteTagDirect(record.flvTag)
			utils.CheckError(err)
		}
		//fmt.Println("rtp seq:", rp.SequenceNumber, ",payload size: ", len(flvTag), ",rtp timestamp: ", rp.Timestamp)

		record.Reset() //重置tag缓存

	}
	return nil
}

func (q *Queue) getDelay() {
	tmpBuf := q.flvRecord.flvTag[4:8]
	ts := uint32(tmpBuf[3])<<24 + uint32(tmpBuf[0])<<16 + uint32(tmpBuf[1])<<8 + uint32(tmpBuf[2])
	if q.startTime == 0 { //还没有从云端获取到初始时间
		utils.UpdatePublishers()
		q.startTime = q.App.Publishers[q.Ssrc].StartTime
		configure.Log.Infof("[ssrc=%v]get stream startTime from cloudserver, startTime=%v", q.Ssrc, q.startTime)
		return
	} else {
		now := time.Now().UnixMilli()
		if delay := int(now - (q.startTime + int64(ts))); delay < 0 {
			return
		} else {
			q.delay = delay
			configure.Log.Tracef("[ssrc=%v]时延：%vms", q.Ssrc, q.delay)
		}
	}
}

func (q *Queue) isFirstOk() bool {
	if rp, ok := q.queue.Get(0); ok {
		if rp != nil {
			return true
		}
	}
	return false
}

func (q *Queue) getHeadNil() int {
	q.m.Lock()
	defer q.m.Unlock()
	nils := 1
	for {
		if val, ok := q.queue.Get(nils); ok {
			if val == nil {
				nils += 1
			} else {
				return nils
			}
		} else {
			return nils
		}
	}
}

func (q *Queue) reshape() int {
	q.m.Lock()
	defer q.m.Unlock()

	removed := 0
	if val, ok := q.queue.Get(1); ok {
		if val == nil { //三个连续的丢包
			quic.GetByQuic(q.Ssrc, q.FirstSeq+1) //让云端知道丢了三个连续包
			for q.queue.Size() > 0 {
				q.queue.Remove(0)
				q.FirstSeq += 1
				q.accLoss += 1
				removed += 1
				if q.isFirstOk() {
					return removed
				}
			}
		}
	}
	return 0
}

func (q *Queue) print() {
	configure.Log.Info("rtp队列长度：", q.queue.Size())
	configure.Log.Info("rtp队列：")
	for i := 0; i < q.queue.Size(); i++ {
		rp, _ := q.queue.Get(i)
		if rp == nil {
			configure.Log.Infof(" nil")
		} else {
			configure.Log.Info(rp.(*rtp.RtpPack).SequenceNumber, " ")
		}
	}
	configure.Log.Infof("\n")

}

func (q *Queue) Close() {
	if q.flvFile != nil {
		q.flvFile.Close()
	}
	configure.Log.Infof("stream closed ssrc=%v", q.Ssrc)
	q.Ssrc = 0 // 表示流已关闭，用于别处判断
	runtime.GC()
}
