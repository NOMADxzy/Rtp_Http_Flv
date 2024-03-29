package quic

import (
	"Rtp_Http_Flv/container/rtp"
	"Rtp_Http_Flv/parser"
	"context"
	"encoding/binary"
	"errors"
	"github.com/quic-go/quic-go"
)

var rtpParser = parser.NewRtpParser()

type Conn struct {
	Connection quic.Connection
	infoStream quic.Stream
	dataStream quic.Stream
}

func newConn(sess quic.Connection, is_server bool) (*Conn, error) {
	quicStream, err := sess.OpenStream()

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
	//return io.ReadFull(c.dataStream,b)
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
	var length uint16
	//读buffer
	err := c.ReadLen(&length)
	if err != nil {
		return err
	}
	if length == 0 {
		return errors.New("RtpCacheNotFound")
	}
	buf := make([]byte, length)
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
