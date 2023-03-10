package main

import (
	"Rtp_Http_Flv/cache"
	"Rtp_Http_Flv/configure"
	"Rtp_Http_Flv/container/rtp"
	"Rtp_Http_Flv/parser"
	"Rtp_Http_Flv/protocol/httpflv"
	"Rtp_Http_Flv/protocol/quic"
	"Rtp_Http_Flv/utils"
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"net"
	"strings"
	"time"
)

type App struct { //边缘节点实体
	RtpQueueMap map[uint32]*cache.Queue
	publishers  map[uint32]*utils.Publisher
	keySsrcMap  map[string]uint32
	flvFiles    *arraylist.List
}

var app *App

//var RtpQueueMap map[uint32]*queue
//var publishers map[uint32]*utils.Publisher
//var keySsrcMap map[string]uint32
//var flvFiles *arraylist.List

func main() {
	if !configure.GetFlag() {
		return
	}
	//初始化一些表
	app = &App{
		publishers:  make(map[uint32]*utils.Publisher),
		keySsrcMap:  make(map[string]uint32),
		RtpQueueMap: make(map[uint32]*cache.Queue),
		flvFiles:    arraylist.New(), //用于关闭打开的文件具柄
	}

	go app.CheckAlive()

	myHttpHandler := &MyHttpHandler{}
	httpflv.StartHTTPFlv(myHttpHandler)
	receiveRtp()
}

func handleNewStream(ssrc uint32) *cache.Queue {
	//更新流源信息
	app.publishers = utils.UpdatePublishers()
	fmt.Println("new stream created ssrc = ", ssrc)

	//设置key和ssrc的映射，以播放flv
	key := app.publishers[ssrc].Key
	app.keySsrcMap[key] = ssrc

	//创建rtp流队列
	channel := strings.SplitN(key, "/", 2)[1] //文件名

	flvRecord := cache.NewFlvRecord()
	flvFile := utils.CreateFlvFile(channel)
	app.flvFiles.Add(flvFile)
	rtpQueue := cache.NewQueue(ssrc, key, configure.RTP_QUEUE_PADDING_WINDOW_SIZE, flvRecord, flvFile)
	app.RtpQueueMap[ssrc] = rtpQueue

	go rtpQueue.RecvPacket()
	go rtpQueue.Play()
	return rtpQueue
}

func handleNewPacket(rp *rtp.RtpPack) {

	rtpQueue := app.RtpQueueMap[rp.SSRC]
	if rtpQueue == nil { //新的ssrc流
		rtpQueue = handleNewStream(rp.SSRC)
	}

	//Rtp包顺序存放到队列中
	//rtpQueue.Enqueue(rp)
	rtpQueue.InChan <- rp

}

type MyHttpHandler struct {
}

func (myHttpHandler *MyHttpHandler) HandleNewFlvWriter(key string, flvWriter *httpflv.FLVWriter) {
	rtpQueue := app.RtpQueueMap[app.keySsrcMap[key]]
	rtpQueue.FlvWriters.Add(flvWriter)
}

func receiveRtp() {

	addr, err := net.ResolveUDPAddr("udp4", configure.UDP_SOCKET_ADDR)
	if err != nil {
		panic(err)
	}

	// Open up a connection
	//conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	defer func() {
		for _, val := range app.flvFiles.Values() {
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
			if err != nil || utils.IsPacketLoss() {
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

func (app *App) CheckAlive() {
	for {
		<-time.After(5 * time.Second) //
		app.publishers = utils.UpdatePublishers()
		for ssrc := range app.RtpQueueMap {
			info := app.publishers[ssrc]
			if info == nil { //流已关闭
				rtpQueue := app.RtpQueueMap[ssrc]
				delete(app.keySsrcMap, rtpQueue.ChannelKey)
				delete(app.RtpQueueMap, rtpQueue.Ssrc)
				rtpQueue.Close()
				quic.CloseQuic()
			}
		}
	}
}
