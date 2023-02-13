package main

import (
	"container/list"
	"fmt"
	flv "github.com/zhangpeihao/goflv"
	"go-mpu/container/rtp"
	"net"
	"os"
	//"go-mpu/container/flv"
	"go-mpu/parser"
)

//var wg sync.WaitGroup

func main() {

	receiveRtp()

}

func receiveRtp() {
	ip := "0.0.0.0:5222"
	udpAddr, err := net.ResolveUDPAddr("udp", ip)
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	f, err := os.Create("./h264toflv.flv") //创建文件
	if err != nil {
		panic(err)
	}
	//w := bufio.NewWriter(f) //创建新的 Writer 对象

	// H264 解析器
	//h264Parser := h264.NewParser()

	// 时间戳计算
	//var cts uint32 = 0

	// 定时器
	//t := time.NewTimer(time.Second * 10)

	flvFile, err := flv.CreateFile("./recv.flv")
	if err != nil {
		fmt.Println("Create FLV dump file error:", err)
		return
	}
	defer func() {
		if flvFile != nil {
			flvFile.Close()
		}
	}()

	var flv_tag []byte
	FlvTagList := list.New()
	rtpQueue := newQueue(10)

	pos := 0
	var last_ts uint32
	last_ts = uint32(0)

	//RtpList := list.New()

	tmpBuf := make([]byte, 4) //读元信息用
	for {
		buff := make([]byte, 2*1024)
		num, err := conn.Read(buff)

		if err != nil {
			continue
		}

		data := buff[:num]
		rtpParser := parser.NewRtpParser()
		rp := rtpParser.Parse(data)
		if rp == nil {
			continue
		}
		//Rtp顺序存放到队列中
		rtpQueue.Enqueue(rp)
		//if rtpQueue.queue.Size()%5 == 0 {
		//	rtpQueue.print()
		//}
		if rtpQueue.queue.Size() < 20 { //刚开始先缓存一定量
			continue
		} else if rtpQueue.queue.Size() == 20 {
			rtpQueue.Check()
			continue
		}
		//到达一定量后就从队列中取rtp了
		rp = rtpQueue.Dequeue().(*rtp.RtpPack)
		payload := rp.Payload
		marker := rp.Marker
		new_ts := rp.Timestamp

		fmt.Println("-----------------", rp.SequenceNumber, "-----------------")
		if int(rp.SequenceNumber)%100 == 0 {
			rtpQueue.print()
		}

		if marker == byte(0) { //该帧未结束
			if new_ts > last_ts { //该帧是初始帧
				// Read tag size
				copy(tmpBuf[1:], payload[1:4])
				TagSize := uint32(tmpBuf[1])<<16 | uint32(tmpBuf[2])<<8 | uint32(tmpBuf[3]) + uint32(11)
				fmt.Println("新建初始帧长度为", TagSize)
				flv_tag = make([]byte, TagSize)

				copy(flv_tag[pos:pos+len(payload)], payload)
				pos += len(payload)
			} else { //该帧是中间帧
				copy(flv_tag[pos:pos+len(payload)], payload)
				pos += len(payload)
			}
		} else { //该帧是结束帧
			if new_ts > last_ts { //没有之前分片
				flv_tag = payload
			} else { //有前面的分片
				//fmt.Println("pos===", pos)
				//fmt.Println(len(payload))
				copy(flv_tag[pos:pos+len(payload)], payload)
			}
			//if flv_tag[0] == byte(9) {
			//	fmt.Println(flv_tag)
			//}
			//得到一个flv tag

			//丢包
			if new_ts%1 == 0 {
				FlvTagList.PushBack(flv_tag)
				flvFile.WriteTagDirect(flv_tag)
			}
			//fmt.Println(flv_tag)
			fmt.Println("rtp seq:", rp.SequenceNumber, ",payload size: ", len(flv_tag), ",rtp timestamp: ", rp.Timestamp)

			flv_tag = nil
			pos = 0

		}
		last_ts = new_ts

		//fmt.Println("seq", rp.SequenceNumber, "  size: ", num)

		// 提取 h.264

		//log.Println(rtpPack)
	}
	f.Close()

}
