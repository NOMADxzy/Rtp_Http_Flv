package quic

import (
	"Rtp_Http_Flv/container/rtp"
	"Rtp_Http_Flv/parser"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
)

var rtpParser = parser.NewRtpParser()

type Conn struct {
	Connection quic.Connection
	infoStream quic.Stream // 存储请求信息，例如 sequence, ssrc
	dataStream quic.Stream // 存储 rtp 数据包信息
}

func newConn(sess quic.Connection, is_server bool) (*Conn, error) {
	quicStream, err := sess.OpenStream()
	if is_server {
		fmt.Print("This is quic server launched by cloudServer.\n")
	}
	if err != nil {
		return nil, err
	}
	return &Conn{
		Connection: sess,
		infoStream: quicStream,
	}, nil
}

//	func (c *conn) DataStream() quic.Stream {
//		return c.dataStream
//	}

func (c *Conn) ReadLen(len *uint16) error {
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
}

func (c *Conn) ReadRtp(pkt **rtp.RtpPack) error {
	if c.dataStream == nil {
		var err error
		c.dataStream, err = c.Connection.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return err
		}
	}
	var rtpLen uint16
	//读buffer
	err := c.ReadLen(&rtpLen)
	if err != nil {
		return err
	}
	if rtpLen == 0 {
		return errors.New("RtpCacheNotFound")
	}
	buf := make([]byte, rtpLen)
	_, err = c.dataStream.Read(buf)
	if err != nil {
		panic(err)
	}
	*pkt = rtpParser.Parse(buf)
	return nil
}

func (c *Conn) WriteSeq(seq uint16) (int, error) {
	seq_b := make([]byte, 2)
	binary.BigEndian.PutUint16(seq_b, seq)
	return c.infoStream.Write(seq_b)
}

func (c *Conn) WriteSsrc(ssrc uint32) (int, error) {
	ssrc_b := make([]byte, 4)
	binary.BigEndian.PutUint32(ssrc_b, ssrc)
	return c.infoStream.Write(ssrc_b)
}
