package core

import "net"

var (
	PUBLISH = "publish"
	PLAY    = "play"
)

type Conn struct {
	net.Conn
	c string
}

func NewConn(c net.Conn) *Conn {
	return &Conn{
		Conn: c,
	}
}
