package main

import (
	"context"
	"encoding/binary"
	"github.com/quic-go/quic-go"
	"rtp_http_flv/container/rtp"
	"rtp_http_flv/parser"
)

var rtpParser = parser.NewRtpParser()

type conn struct {
	session    quic.Connection
	infoStream quic.Stream
	dataStream quic.Stream
}

func newConn(sess quic.Connection, isServer bool) (*conn, error) {
	var quicStream quic.Stream
	var err error
	if isServer {
		quicStream, err = sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			session:    sess,
			dataStream: quicStream,
		}, nil
	} else {
		quicStream, err := sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			session:    sess,
			infoStream: quicStream,
		}, nil
	}
}

//	func (c *conn) DataStream() quic.Stream {
//		return c.dataStream
//	}

func (c *conn) ReadLen(len *uint16) error {
	if c.infoStream == nil {
		var err error
		c.dataStream, err = c.session.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return err
		}
	}
	lenB := make([]byte, 2)
	_, err := c.infoStream.Read(lenB)
	if err != nil {
		return err
	}
	*len = binary.BigEndian.Uint16(lenB)
	return nil
	//return io.ReadFull(c.dataStream,b)
}

func (c *conn) ReadRtp(pkt **rtp.RtpPack) error {
	if c.dataStream == nil {
		var err error
		c.dataStream, err = c.session.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return err
		}
	}
	var bufLen uint16
	//读buffer
	err := c.ReadLen(&bufLen)
	if err != nil {
		return err
	}
	buf := make([]byte, bufLen)
	_, err = c.dataStream.Read(buf)
	if err != nil {
		panic(err)
	}
	*pkt = rtpParser.Parse(buf)
	return nil
}

func (c *conn) WriteSeq(seq uint16) (int, error) {
	seqB := make([]byte, 2)
	binary.BigEndian.PutUint16(seqB, seq)
	return c.infoStream.Write(seqB)
}
