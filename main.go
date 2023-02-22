package main

import (
	"fmt"
	"net"
	"rtp_http_flv/container/rtp"
	"rtp_http_flv/parser"
	"time"
)

type FlvRecord struct {
	HttpServer *Server
	flvTag     []byte
	TagSize    int
	pos        int
	lastTs     uint32
	lostTs     uint32
}

func main() {
	receiveRtp()
}

func receiveRtp() {
	address := "0.0.0.0:5222"

	addr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		panic(err)
	}

	// Open up a connection
	//connRtp, err := net.ListenMulticastUDP("udp4", nil, addr)
	connRtp, _ := net.ListenUDP("udp", addr)

	flvFile, err := CreateFile("./recv.flv")
	if err != nil {
		fmt.Println("Create FLV dump file error:", err)
		return
	}
	defer func() {
		if flvFile != nil {
			flvFile.Close()
		}
	}()

	rtpQueue := newQueue(10)
	flvRecord := &FlvRecord{
		startHTTPFlv(), nil, 0, 0, uint32(0), 0,
	}

	go func() { // 接受rtp协程
		for {
			// 读udp数据
			buff := make([]byte, 2*1024)
			//num, err := connRtp.Read(buff)
			num, _, err := connRtp.ReadFromUDP(buff)
			if err != nil {
				continue
			}

			// 解析为rtp包
			data := buff[:num]
			rtpParser := parser.NewRtpParser()
			rp := rtpParser.Parse(data)
			if rp == nil {
				continue
			}

			// Rtp 包顺序存放到队列中
			rtpQueue.Enqueue(rp)

			if rtpQueue.queue.Size() < 2*rtpQueue.PaddingWindowSize { // 刚开始先缓存一定量
				continue
			} else if !rtpQueue.checked {
				fmt.Println("rtp队列进行check")
				rtpQueue.Check()
				continue
			}
			// 到达一定量后就从队列中取rtp了
			//if !rtpQueue.reading {
			//	go rtpQueue.offerPacket()
			//}
			for {
				protoRp := rtpQueue.Dequeue() // 阻塞
				err = extractFlv(protoRp, flvRecord, rtpQueue, flvFile, false)
				if err != nil {
					fmt.Println(err)
					flvRecord.pos = 0
					flvRecord.flvTag = nil
				}
				if rtpQueue.queue.Size() == rtpQueue.PaddingWindowSize*2 {
					break
				}
			}

		}
	}()

	//for {
	//	proto_rp, ok := <-rtpQueue.readChan
	//	if ok {
	//
	//		err = extractFlv(proto_rp, flvRecord, rtpQueue, flvFile, false)
	//		if err != nil {
	//			panic(err)
	//	}
	//	}
	//}
	time.Sleep(time.Hour)

}

// 从rtp包中提取出flv_tag，根据record信息组合分片，debug打印调试信息
func extractFlv(protoRp interface{}, record *FlvRecord, rtpQueue *queue, flvFile *File, debug bool) error {
	if protoRp == nil {
		record.flvTag = nil
		record.pos = 0
		return nil
	}
	rp := protoRp.(*rtp.RtpPack)

	//
	payload := rp.Payload
	marker := rp.Marker
	new_ts := rp.Timestamp

	tmpBuf := make([]byte, 4)
	if debug {
		fmt.Println("-----------------", rp.SequenceNumber, "-----------------")
	}
	if int(rp.SequenceNumber)%100 == 0 {
		rtpQueue.print()
	}

	if marker == byte(0) { //该帧未结束
		if record.flvTag == nil { //该帧是初始帧
			if !isTagHead(payload) {
				fmt.Println("错误，非法的tag头")
				return nil
			}
			// Read tag size
			copy(tmpBuf[1:], payload[1:4])
			record.TagSize = int(uint32(tmpBuf[1])<<16 | uint32(tmpBuf[2])<<8 | uint32(tmpBuf[3]) + uint32(11))
			fmt.Println("新建初始帧长度为", record.TagSize)
			record.flvTag = make([]byte, record.TagSize)

			copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
			record.pos += len(payload)
		} else { //该帧是中间帧
			if record.pos+len(payload) < record.TagSize {
				copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
				record.pos += len(payload)
			} else { //发生了丢包
				record.flvTag = nil
				record.pos = 0
				return nil
			}
		}
	} else { //该帧是结束帧
		if record.flvTag == nil { //没有之前分片
			if isTagHead(payload) {
				record.flvTag = payload
			} else {
				return nil
			}
		} else { //有前面的分片
			//fmt.Println("pos===", pos)
			//fmt.Println(len(payload))
			if record.pos+len(payload) == record.TagSize {
				copy(record.flvTag[record.pos:record.pos+len(payload)], payload)
			} else { //这个tag不完整了
				record.flvTag = nil
				record.pos = 0
				return nil
			}
		}
		//得到一个flv tag

		//有客户端就将flv数据发给客户端
		if record.HttpServer.flvWriter != nil {
			//FlvTagList.PushBack(flv_tag)
			err := record.HttpServer.flvWriter.Write(record.flvTag)
			if err != nil {
				return err
			}
		}
		//录制到文件中
		err := flvFile.WriteTagDirect(record.flvTag)
		if err != nil {
			return err
		}
		//fmt.Println("rtp seq:", rp.SequenceNumber, ",payload size: ", len(flv_tag), ",rtp timestamp: ", rp.Timestamp)

		record.flvTag = nil
		record.pos = 0

	}
	record.lastTs = new_ts
	return nil
}

func isTagHead(payload []byte) bool {
	if payload[0] == byte(8) || payload[0] == byte(9) {
		if payload[8] == byte(0) && payload[9] == byte(0) && payload[10] == byte(0) {
			return true
		}
	}
	return false
}
