package main

import (
	"Rtp_Http_Flv/cache"
	"Rtp_Http_Flv/configure"
	"Rtp_Http_Flv/container/rtp"
	"Rtp_Http_Flv/parser"
	"Rtp_Http_Flv/protocol/httpflv"
	"Rtp_Http_Flv/utils"
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/q191201771/pprofplus/pprofplus/pkg/pprofplus"
	"net"
	"strings"
)

var app *cache.App

func main() {
	if !configure.GetFlag() {
		return
	}

	defer func() {
		for _, val := range app.FlvFiles.Values() {
			flvFile := val.(*utils.File)
			flvFile.Close()
		}
	}()
	//初始化一些表
	app = &cache.App{
		Publishers:    make(map[uint32]*utils.Publisher),
		KeySsrcMap:    make(map[string]uint32),
		RtpQueueMap:   make(map[uint32]*cache.Queue),
		UdpBufferSize: 100 * 1024,      //udp socket的缓存大小，初始设为100KB
		FlvFiles:      arraylist.New(), //用于关闭打开的文件具柄
	}

	err := pprofplus.Start() //内存监测
	utils.CheckError(err)

	go app.CheckAlive() //检验流是否关闭

	myHttpHandler := &MyHttpHandler{}
	httpflv.StartHTTPFlv(myHttpHandler) //开启httpflv服务

	receiveRtp() //收rtp包
}

func handleNewStream(ssrc uint32) *cache.Queue {
	//更新流源信息
	app.Publishers = utils.UpdatePublishers()
	fmt.Println("new stream created ssrc = ", ssrc)

	//设置key和ssrc的映射，以播放flv
	key := app.Publishers[ssrc].Key
	startTime := app.Publishers[ssrc].StartTime
	app.KeySsrcMap[key] = ssrc

	//创建rtp流队列
	channel := strings.SplitN(key, "/", 2)[1] //文件名

	flvRecord := cache.NewFlvRecord()

	var flvFile *utils.File
	if configure.ENABLE_RECORD {
		fmt.Println("Create record file path = ", configure.RECORD_DIR, "/", channel+".flv")
		flvFile := utils.CreateFlvFile(channel)
		app.FlvFiles.Add(flvFile)
	}

	rtpQueue := cache.NewQueue(ssrc, key, configure.RTP_QUEUE_PADDING_WINDOW_SIZE, flvRecord, flvFile, startTime, app)
	app.RtpQueueMap[ssrc] = rtpQueue

	go rtpQueue.RecvPacket()
	//go rtpQueue.Play()
	return rtpQueue
}

func handleNewPacket(rp *rtp.RtpPack) {

	rtpQueue := app.RtpQueueMap[rp.SSRC]
	if rtpQueue == nil { //新的ssrc流
		rtpQueue = handleNewStream(rp.SSRC)
	}
	//Rtp包顺序存放到队列中
	rtpQueue.InChan <- rp

}

type MyHttpHandler struct {
}

func (myHttpHandler *MyHttpHandler) HandleNewFlvWriter(key string, flvWriter *httpflv.FLVWriter) {
	rtpQueue := app.RtpQueueMap[app.KeySsrcMap[key]]
	rtpQueue.FlvWriters.Add(flvWriter)
}

func receiveRtp() {

	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0"+configure.UDP_SOCKET_ADDR)
	utils.CheckError(err)
	fmt.Printf("Udp Socket listen On 0.0.0.0%v\n", configure.UDP_SOCKET_ADDR)

	// Open up a connection
	//conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	conn, err := net.ListenUDP("udp", addr)
	utils.CheckError(err)
	err = conn.SetReadBuffer(app.UdpBufferSize)
	utils.CheckError(err)

	app.UdpConn = conn
	rtpParser := parser.NewRtpParser()

	for {
		//读udp数据
		buff := make([]byte, 1300)
		//num, err := conn.Read(buff)
		num, _, err := conn.ReadFromUDP(buff)
		utils.CheckError(err)

		if utils.IsPacketLoss() {
			continue
		}

		//解析为rtp包
		data := buff[:num]
		rp := rtpParser.Parse(data)
		if rp != nil {
			handleNewPacket(rp)
		}
		utils.CheckError(err)
	}

}
