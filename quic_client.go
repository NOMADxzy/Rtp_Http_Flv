package main

import (
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"go-mpu/container/rtp"
)

//var Conn = initQuic()

func initQuic() *conn {
	tlsConf := &tls.Config{InsecureSkipVerify: true,
		NextProtos: []string{"quic-echo-server"}}
	protoconn, err := quic.DialAddr("localhost:4242", tlsConf, nil)
	if err != nil {
		panic(err)
	}
	conn, _ := newConn(protoconn, false)
	return conn
}

func GetByQuic(seq uint16, rtpQueue *queue, Conn *conn) {
	// run the client
	go func() {

		// 根据序列号请求
		_, err := Conn.WriteSeq(uint16(seq))

		//读rtp数据
		var pkt *rtp.RtpPack
		_, err = Conn.ReadRtp(&pkt)
		if err != nil {
			fmt.Println(err)
		}
		rtpQueue.Enqueue(pkt)
		//fmt.Printf("buf:\t %v \n", pkt.buffer)
		//fmt.Printf("ekt:\t %v \n", pkt.ekt)
		//fmt.Println("buf len:", len(pkt.))
		fmt.Printf("Seq:\t %v \n", pkt.SequenceNumber)
		fmt.Printf("SSRC:\t %v \n", pkt.SSRC)
		//fmt.Printf("TimeStamp:\t %v \n", pkt.GetTimestamp())
		//fmt.Printf("ExtLen:\t %v \n", pkt.GetHdrExtLen())
		//fmt.Printf("PTtype:\t %v \n", pkt.GetPT())
		fmt.Printf("Payload:\t %v \n", pkt.Payload)
	}()
}

//func main() {
//	//GetByQuic(uint16(4300))
//	time.Sleep(time.Hour)
//}
