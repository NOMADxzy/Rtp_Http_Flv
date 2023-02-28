package main

import (
	"github.com/emirpasic/gods/lists/arraylist"
	"go-mpu/configure"
	"go-mpu/container/rtp"
	"go-mpu/parser"
	"go-mpu/utils"
	"net"
	"strings"
	"time"
)

var RtpQueueMap map[uint32]*queue
var publishers map[uint32]*utils.Publisher
var keySsrcMap map[string]uint32
var flvFiles *arraylist.List

type FlvRecord struct {
	flvTag         []byte //记录当前flvTag写入的字节情况
	TagSize        int
	pos            int  //写入flvTag的位置
	jumpToNextHead bool //发生丢包后跳到下一个tag头开始解析
}

func (flvRecord *FlvRecord) Reset() {
	flvRecord.flvTag = nil
	flvRecord.TagSize = 0
	flvRecord.pos = 0
}

func main() {
	startHTTPFlv()
	receiveRtp()
}

func handleNewPacket(rp *rtp.RtpPack) {

	if RtpQueueMap == nil {
		RtpQueueMap = make(map[uint32]*queue)
	}
	rtpQueue := RtpQueueMap[rp.SSRC]
	if rtpQueue == nil { //新的ssrc流

		//更新流源信息
		if publishers == nil {
			publishers = make(map[uint32]*utils.Publisher)
		}
		utils.UpdatePublishers(publishers)

		//设置key和ssrc的映射，以播放flv
		if keySsrcMap == nil {
			keySsrcMap = make(map[string]uint32)
		}
		key := publishers[rp.SSRC].Key
		keySsrcMap[key] = rp.SSRC

		//创建rtp流队列
		channel := strings.SplitN(key, "/", 2)[1] //文件名

		flvRecord := &FlvRecord{
			nil, 0, 0, false,
		}
		flvFile := utils.CreateFlvFile(channel)
		flvFiles.Add(flvFile)
		rtpQueue = newQueue(rp.SSRC, configure.RTP_QUEUE_PADDING_WINDOW_SIZE, flvRecord, flvFile)
		RtpQueueMap[rp.SSRC] = rtpQueue
	}

	//Rtp包顺序存放到队列中
	rtpQueue.Enqueue(rp)

	if rtpQueue.queue.Size() < 2*rtpQueue.PaddingWindowSize { //刚开始先缓存一定量
		return
	} else if !rtpQueue.checked {
		rtpQueue.Check()
		return
	}
	//到达一定量后就从队列中取rtp了
	rtpQueue.Play()

}

func HandleNewFlvWriter(key string, flvWriter *FLVWriter) {
	rtpQueue := RtpQueueMap[keySsrcMap[key]]
	rtpQueue.flvWriters.Add(flvWriter)
}

func receiveRtp() {

	addr, err := net.ResolveUDPAddr("udp4", configure.UDP_SOCKET_ADDR)
	if err != nil {
		panic(err)
	}

	// Open up a connection
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	//conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	flvFiles = arraylist.New() //用于关闭打开的文件具柄
	defer func() {
		for _, val := range flvFiles.Values() {
			flvFile := val.(*utils.File)
			flvFile.Close()
		}
	}()

	go func() { //接受rtp协程
		for {
			//读udp数据
			buff := make([]byte, 2*1024)
			//num, err := conn.Read(buff)
			num, _, err := conn.ReadFromUDP(buff)
			if err != nil {
				continue
			}

			//解析为rtp包
			data := buff[:num]
			rtpParser := parser.NewRtpParser()
			rp := rtpParser.Parse(data)
			if rp == nil {
				continue
			}

			handleNewPacket(rp)

		}
	}()

	time.Sleep(time.Hour)

}

//从rtp包中提取出flvTag，根据record信息组合分片，debug打印调试信息
func extractFlv(protoRp interface{}, rtpQueue *queue) error {
	record := rtpQueue.flvRecord

	if protoRp == nil { //该包丢失了
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
			rtpQueue.cache.Write(record.flvTag)
		}
		for i := 0; i < rtpQueue.flvWriters.Size(); i++ {
			val, f := rtpQueue.flvWriters.Get(i)
			if f {
				writer := val.(*FLVWriter)
				if writer.closed {
					rtpQueue.flvWriters.Remove(i)
				} else { //播放该分段
					if !writer.init {
						err := rtpQueue.cache.SendInitialSegment(writer)
						if err != nil {
							return err
						}
						writer.init = true
					} else {
						writer.Write(record.flvTag)
					}
				}
			}
		}

		//录制到文件中
		err := rtpQueue.flvFile.WriteTagDirect(record.flvTag)
		if err != nil {
			return err
		}
		//fmt.Println("rtp seq:", rp.SequenceNumber, ",payload size: ", len(flvTag), ",rtp timestamp: ", rp.Timestamp)

		record.flvTag = nil
		record.pos = 0

	}
	return nil
}
