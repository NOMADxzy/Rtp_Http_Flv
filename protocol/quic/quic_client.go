package quic

import (
	"Rtp_Http_Flv/configure"
	"Rtp_Http_Flv/container/rtp"
	"Rtp_Http_Flv/utils"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/sirupsen/logrus"
	"sync"
)

var QuicConn *Conn
var lock sync.Mutex

//var Conn *conn

func initQuic() *Conn {
	tlsConf := &tls.Config{InsecureSkipVerify: true,
		NextProtos: []string{"quic-echo-server"}}
	protoconn, err := quic.DialAddr(configure.Conf.CLOUD_HOST+configure.Conf.QUIC_ADDR, tlsConf, nil)
	if err != nil {
		panic(err)
	}
	conn, _ := newConn(protoconn, false)
	return conn
}

func CloseQuic() {
	if QuicConn == nil {
		return
	}
	if QuicConn.dataStream != nil {
		err := QuicConn.dataStream.Close()
		utils.CheckError(err)
	}
	if QuicConn.infoStream != nil {
		err := QuicConn.infoStream.Close()
		utils.CheckError(err)
	}
	QuicConn = nil
	configure.Log.Infof("quic conn closed\n")
}

func GetByQuic(ssrc uint32, seq uint16) *rtp.RtpPack {
	lock.Lock() //防止多条流同时调用重传导致出错
	defer lock.Unlock()

	if configure.Conf.DISABLE_QUIC {
		return nil
	}

	configure.Log.WithFields(logrus.Fields{
		"seq": seq,
	}).Debugf("[ssrc=%v] packet lost, run quic request", ssrc)
	if QuicConn == nil {
		QuicConn = initQuic()
	}
	// run the client
	// 根据序列号请求
	_, err := QuicConn.WriteSsrc(ssrc)
	if err != nil { //长时间未重传，导致服务关闭
		configure.Log.Error(err)
		QuicConn = initQuic()
		_, err = QuicConn.WriteSsrc(ssrc)
	}

	_, err = QuicConn.WriteSeq(seq)

	//读rtp数据
	var pkt *rtp.RtpPack
	err = QuicConn.ReadRtp(&pkt)
	if err != nil {
		//没有该包的缓存
		configure.Log.Errorf("[ssrc=%v]quic err, get packet failed, seq=%v\n", ssrc, seq)
		return nil
	}
	if pkt == nil {
		configure.Log.Errorf("[ssrc=%v]quic err，received a nil packet, seq=%v\n", ssrc, seq)
	} else {
		configure.Log.WithFields(logrus.Fields{
			"seq": pkt.SequenceNumber,
		}).Debugf("quic received rtp packet")
		return pkt
	}
	return nil
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
