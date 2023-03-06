package main

import (
	"Rtp_Http_Flv/container/rtp"
	"Rtp_Http_Flv/parser"
	"context"
	"encoding/binary"
	"errors"
	"github.com/quic-go/quic-go"
)

var rtpParser = parser.NewRtpParser()

type conn struct {
	Connection quic.Connection
	infoStream quic.Stream
	dataStream quic.Stream
}

func newConn(sess quic.Connection, is_server bool) (*conn, error) {
	if is_server {
		dstream, err := sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			Connection: sess,
			dataStream: dstream,
		}, nil
	} else {
		istream, err := sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			Connection: sess,
			infoStream: istream,
		}, nil
	}
}

//	func (c *conn) DataStream() quic.Stream {
//		return c.dataStream
//	}
func (c *conn) ReadLen(len *uint16) error {
	if c.infoStream == nil {
		var err error
		c.dataStream, err = c.Connection.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return err
		}
	}
	len_b := make([]byte, 2)
	_, err := c.infoStream.Read(len_b)
	if err != nil {
		return err
	}
	*len = binary.BigEndian.Uint16(len_b)
	return nil
	//return io.ReadFull(c.dataStream,b)
}
func (c *conn) ReadRtp(pkt **rtp.RtpPack) error {
	if c.dataStream == nil {
		var err error
		c.dataStream, err = c.Connection.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return err
		}
	}
	var len uint16
	//读buffer
	err := c.ReadLen(&len)
	if err != nil {
		return err
	}
	if len == 0 {
		return errors.New("RtpCacheNotFound")
	}
	buf := make([]byte, len)
	_, err = c.dataStream.Read(buf)
	if err != nil {
		panic(err)
	}
	*pkt = rtpParser.Parse(buf)
	return nil
}

func (c *conn) WriteSeq(seq uint16) (int, error) {
	seq_b := make([]byte, 2)
	binary.BigEndian.PutUint16(seq_b, seq)
	return c.infoStream.Write(seq_b)
}

func (c *conn) WriteSsrc(ssrc uint32) (int, error) {
	ssrc_b := make([]byte, 4)
	binary.BigEndian.PutUint32(ssrc_b, ssrc)
	return c.infoStream.Write(ssrc_b)
}
