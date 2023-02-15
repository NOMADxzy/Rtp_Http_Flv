package rtmp

import (
	"fmt"
	"log"
	"net"
	"strings"
	"testing"

	neturl "net/url"
)

func TestRtmp(t *testing.T) {
	url := "rtmp://127.0.0.0:1935/live/xxxxxxx?aa=ab"
	u, err := neturl.Parse(url)
	if err != nil {
		panic(err)
	}

	fmt.Println(u.Scheme)
	fmt.Println(u.Path)
	fmt.Println(u.Host)
	fmt.Println(u.RawQuery)
	m, _ := neturl.ParseQuery(u.RawQuery)
	fmt.Println(m)
	fmt.Println(m["aa"][0])
	fmt.Println("----------------------------")

	fmt.Println(strings.Index(u.Host, ":"))

	port := ":1935"
	host := u.Host
	//localIP := ":0"
	if strings.Index(host, ":") != -1 {
		host, port, err = net.SplitHostPort(host)
		if err != nil {
			panic(err)
		}
		port = ":" + port
	}

	fmt.Println(host)
	fmt.Println(port)

	ips, err := net.LookupIP(host)
	fmt.Printf("ips: %v, host: %v", ips, host)

	//net.DialTCP()

}

func TestRtmp2(t *testing.T) {

	ips, _ := net.LookupIP("baidu.com")
	for _, ns := range ips {
		fmt.Println(ns)
	}

	fmt.Println(ips)

	rtmpClient := &Client{}
	// 连接 推流
	err := rtmpClient.Dial("rtmp://localhost/live/rfBd56ti2SMtYvSgD5xAV0YU99zampta7Z7S575KLkIZ9PYk", PUBLISH)
	if err != nil {
		log.Panicln(err)
	}

}
