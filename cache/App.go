package cache

import (
	"Rtp_Http_Flv/protocol/quic"
	"Rtp_Http_Flv/utils"
	"github.com/emirpasic/gods/lists/arraylist"
	"net"
	"time"
)

type App struct { //边缘节点实体
	RtpQueueMap   map[uint32]*Queue
	Publishers    map[uint32]*utils.Publisher
	KeySsrcMap    map[string]uint32
	UdpConn       *net.UDPConn
	UdpBufferSize int
	FlvFiles      *arraylist.List
}

func (app *App) CheckAlive() {
	for {
		<-time.After(5 * time.Second) //
		app.Publishers = utils.UpdatePublishers()
		for ssrc := range app.RtpQueueMap {
			info := app.Publishers[ssrc]
			if info == nil { //流已关闭
				rtpQueue := app.RtpQueueMap[ssrc]
				delete(app.KeySsrcMap, rtpQueue.ChannelKey)
				delete(app.RtpQueueMap, rtpQueue.Ssrc)
				rtpQueue.Close()
				quic.CloseQuic()
			}
		}
	}
}
