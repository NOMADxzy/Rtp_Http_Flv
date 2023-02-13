package main

import (
	"context"
	"encoding/binary"
	"github.com/lucas-clemente/quic-go"
	"go-mpu/container/rtp"
	"go-mpu/parser"
)

var rtpParser = parser.NewRtpParser()

type conn struct {
	session    quic.Session
	infoStream quic.Stream
	dataStream quic.Stream
}

func newConn(sess quic.Session, is_server bool) (*conn, error) {
	if is_server {
		dstream, err := sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			session:    sess,
			dataStream: dstream,
		}, nil
	} else {
		istream, err := sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			session:    sess,
			infoStream: istream,
		}, nil
	}
}

//func (c *conn) DataStream() quic.Stream {
//	return c.dataStream
//}
func (c *conn) ReadLen(len *uint16) (int, error) {
	if c.infoStream == nil {
		var err error
		c.dataStream, err = c.session.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return 0, err
		}
	}
	len_b := make([]byte, 2)
	_, err := c.infoStream.Read(len_b)
	if err != nil {
		panic(err)
	}
	*len = binary.BigEndian.Uint16(len_b)
	return 0, nil
	//return io.ReadFull(c.dataStream,b)
}
func (c *conn) ReadRtp(pkt **rtp.RtpPack) (int, error) {
	if c.dataStream == nil {
		var err error
		c.dataStream, err = c.session.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return 0, err
		}
	}
	var len uint16
	//读buffer
	_, err := c.ReadLen(&len)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, len)
	_, err = c.dataStream.Read(buf)
	if err != nil {
		panic(err)
	}
	*pkt = rtpParser.Parse(buf)
	return 0, nil
}

func (c *conn) WriteSeq(seq uint16) (int, error) {
	seq_b := make([]byte, 2)
	binary.BigEndian.PutUint16(seq_b, seq)
	return c.infoStream.Write(seq_b)
}
