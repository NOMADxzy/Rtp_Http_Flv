package core

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	neturl "net/url"
	"strings"
)

type ConnClient struct {
	conn *Conn

	transId  int
	url      string
	tcUrl    string
	app      string
	title    string
	query    string
	streamId uint32
}

func NewConnClient() *ConnClient {
	return &ConnClient{
		conn:     nil,
		transId:  0,
		url:      "",
		tcUrl:    "",
		app:      "",
		title:    "",
		query:    "",
		streamId: 0,
	}
}

// Rtmp 客户端开始连接服务端
func (connClient *ConnClient) Start(url, method string) (err error) {
	// 1. 解析 url 地址
	u, err := neturl.Parse(url)
	if err != nil {
		return err
	}
	connClient.url = url
	path := strings.TrimLeft(u.Path, "/")
	ps := strings.SplitN(path, "/", 2) // 获取 rtmp 中 AppName
	if len(ps) != 2 {
		return fmt.Errorf("u path err: %s", path)
	}
	connClient.app = ps[0]
	connClient.title = ps[1]
	connClient.query = u.RawQuery
	connClient.tcUrl = "rtmp://" + u.Host + "/" + connClient.app

	// 2. 解析 本地 Ip 和 服务端 Ip 地址
	port := ":1935"
	host := u.Host
	localIP := ":0"
	var remoteIP string
	if strings.Index(host, ":") != -1 {
		host, port, err = net.SplitHostPort(host)
		if err != nil {
			return err
		}
		port = ":" + port
	}

	// 3. 通过域名查询 IPv4 和 IPv6 信息, 获取服务端 ip 地址
	ips, err := net.LookupIP(host)
	log.Printf("ips: %v, host: %v \n", ips, host)
	if err != nil {
		//log.Warning(err)
		fmt.Println(err)

		return err
	}
	remoteIP = ips[rand.Intn(len(ips))].String()
	if strings.Index(remoteIP, ":") == -1 {
		remoteIP += port
	}

	// 4. 获取 localIP 的 TCPAddr
	log.Println("localIP: ", localIP)
	local, err := net.ResolveTCPAddr("tcp", localIP)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 5. 获取 remoteIP 的 TCPAddr
	log.Println("remoteIP: ", remoteIP)
	remote, err := net.ResolveTCPAddr("tcp", remoteIP)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 6. tcp 连接
	conn, err := net.DialTCP("tcp", local, remote)
	if err != nil {
		fmt.Println(err)
		return err
	}

	log.Println("connection:", "local:", conn.LocalAddr(), "remote:", conn.RemoteAddr())

	//connClient.conn = NewConn(conn, 4*1024)

	log.Println("HandshakeClient....")

	return
}
