package main

import (
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"rtp_http_flv/container/rtp"
	"sync"
)

// var Conn = initQuic()
var lock sync.Mutex

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

func (q *queue) GetByQuic(seq uint16) {
	lock.Lock()
	defer lock.Unlock()

	Conn := q.Conn
	// run the client
	// 根据序列号请求
	_, err := Conn.WriteSeq(uint16(seq))

	//读rtp数据
	var pkt *rtp.RtpPack
	err = Conn.ReadRtp(&pkt)
	if err != nil {
		//没有该包的缓存
		fmt.Println("错误，quic无法获取包,序号：", seq)
		return
	}
	if pkt == nil {
		fmt.Println("错误，quic收到一个空包")
	} else {
		q.Enqueue(pkt)
		fmt.Printf("quic收到rtp包，Seq:\t %v \n", pkt.SequenceNumber)
	}
	//fmt.Printf("buf:\t %v \n", pkt.buffer)
	//fmt.Printf("ekt:\t %v \n", pkt.ekt)
	//fmt.Println("buf len:", len(pkt.))
	//fmt.Printf("SSRC:\t %v \n", pkt.SSRC)
	//fmt.Printf("TimeStamp:\t %v \n", pkt.GetTimestamp())
	//fmt.Printf("ExtLen:\t %v \n", pkt.GetHdrExtLen())
	//fmt.Printf("PTtype:\t %v \n", pkt.GetPT())
	//fmt.Printf("Payload:\t %v \n", pkt.Payload)
}

//func main() {
//	//GetByQuic(uint16(4300))
//	time.Sleep(time.Hour)
//}
