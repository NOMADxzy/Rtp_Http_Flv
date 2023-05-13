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
	"net"
	"strconv"
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

	//err := pprofplus.Start() //内存监测
	//utils.CheckError(err)

	go app.CheckAlive() //检验流是否关闭

	myHttpHandler := &MyHttpHandler{}
	httpflv.StartHTTPFlv(myHttpHandler) //开启httpflv服务

	receiveRtp() //收rtp包
}

func handleNewStream(ssrc uint32) *cache.Queue {
	//更新流源信息
	app.Publishers = utils.UpdatePublishers()
	fmt.Printf("new stream created ssrc=%v\n", ssrc)

	//设置key和ssrc的映射，以播放flv
	key := app.Publishers[ssrc].Key
	app.KeySsrcMap[key] = ssrc

	//创建rtp流队列
	channel := strings.SplitN(key, "/", 2)[1] //文件名

	flvRecord := cache.NewFlvRecord()

	var flvFile *utils.File
	if configure.ENABLE_RECORD {
		fmt.Printf("Create record file path=%v\n", configure.RECORD_DIR+"/"+channel+".flv")
		flvFile = utils.CreateFlvFile(channel)
		app.FlvFiles.Add(flvFile)
	}

	rtpQueue := cache.NewQueue(ssrc, key, configure.RTP_QUEUE_PADDING_WINDOW_SIZE, flvRecord, flvFile, app)
	app.RtpQueueMap[ssrc] = rtpQueue

	go rtpQueue.RecvPacket()
	go rtpQueue.PrintInfo()
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

func (myHttpHandler *MyHttpHandler) HandleNewFlvWriterRequest(key string, flvWriter *httpflv.FLVWriter) {
	ssrc := app.KeySsrcMap[key]
	if ssrc == 0 { //定义有效的ssrc不为0

	}
	rtpQueue := app.RtpQueueMap[ssrc]
	rtpQueue.FlvWriters.Add(flvWriter)
}

func (myHttpHandler *MyHttpHandler) HandleDelayRequest(key string) (int64, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("request startTime err, no such key: %s\n", key)
		}
	}()
	startTime := app.Publishers[app.KeySsrcMap[key]].StartTime
	return startTime, nil
}

func (myHttpHandler *MyHttpHandler) HasChannel(path string) bool {
	return app.KeySsrcMap[path] != 0 //有效的ssrc不为0
}

func receiveRtp() {
	var err error
	var addr *net.UDPAddr
	var conn *net.UDPConn

	isMulticast := strings.IndexByte(configure.UDP_SOCKET_ADDR, '.') > 0

	if isMulticast { // 组播
		addr, err = net.ResolveUDPAddr("udp4", configure.UDP_SOCKET_ADDR)
		utils.CheckError(err)
		conn, err = net.ListenMulticastUDP("udp4", nil, addr)
		utils.CheckError(err)
		fmt.Printf("Udp Socket listen On %v\n", configure.UDP_SOCKET_ADDR)

	} else { // 单播
		addr, err = net.ResolveUDPAddr("udp4", "0.0.0.0"+configure.UDP_SOCKET_ADDR)
		utils.CheckError(err)
		conn, err = net.ListenUDP("udp", addr)
		utils.CheckError(err)
		fmt.Printf("Udp Socket listen On 0.0.0.0%v\n", configure.UDP_SOCKET_ADDR)
	}

	err = conn.SetReadBuffer(app.UdpBufferSize)
	utils.CheckError(err)

	app.UdpConn = conn
	rtpParser := parser.NewRtpParser()

	firstPkt := true
	for {
		//读udp数据
		buff := make([]byte, 1300)
		//num, err := conn.Read(buff)
		num, addr, err := conn.ReadFromUDP(buff)
		utils.CheckError(err)
		if firstPkt { // 收到云端的第一个数据包
			firstPkt = false
			configure.CLOUD_HOST = addr.IP.String()
			fmt.Printf("udp connection established, remote IP=%v\n", configure.CLOUD_HOST)
			if buff[0] == '0' && buff[1] == '0' && buff[2] == '0' && buff[3] == '1' { // 标志收到初始化信息
				QuicPort := uint16(buff[4])<<8 + uint16(buff[5]) // 大端地址
				ApiPort := uint16(buff[6])<<8 + uint16(buff[7])  // 大端地址
				fmt.Printf("[logging] initial port message received, quic port=%v, http port=%v\n", QuicPort, ApiPort)

				configure.QUIC_ADDR = ":" + strconv.Itoa(int(QuicPort)) // 优先级：从云端收到的端口初始化信息>本地configure配置的端口初始化信息
				configure.API_ADDR = ":" + strconv.Itoa(int(ApiPort))
				configure.INIT = true

				continue
			} else {
				fmt.Printf("[logging] use local port config, quic port=%v, http port=%v\n", configure.QUIC_ADDR, configure.API_ADDR)
			}
		}

		if utils.IsPacketLoss() {
			continue
		}

		//解析为rtp包
		data := buff[:num]
		rp := rtpParser.Parse(data)
		if rp != nil {
			handleNewPacket(rp)
		}
	}

}
