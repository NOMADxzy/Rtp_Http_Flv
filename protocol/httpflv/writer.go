package httpflv

import (
	"Rtp_Http_Flv/utils"
	"errors"
	"fmt"
	"log"
	"net/http"
)

const (
	headerLen   = 11
	maxQueueNum = 10240
)

type FLVWriter struct {
	Uid             string
	app, title, url string
	buf             []byte
	Closed          bool
	closedChan      chan struct{}
	ctx             http.ResponseWriter
	packetQueue     chan []byte
	Init            bool
}

func NewFLVWriter(app, title, url string, ctx http.ResponseWriter) *FLVWriter {
	ret := &FLVWriter{
		Uid:         "测试id-fjaefiagrklvg",
		app:         app,
		title:       title,
		url:         url,
		ctx:         ctx,
		closedChan:  make(chan struct{}),
		buf:         make([]byte, headerLen),
		packetQueue: make(chan []byte, maxQueueNum),
	}

	ret.ctx.Write([]byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09})
	utils.PutI32BE(ret.buf[:4], 0)
	ret.ctx.Write(ret.buf[:4])
	go func() {

		err := ret.SendPacket()
		if err != nil {
			log.Println("SendPacket error:", err)
			ret.Closed = true
		}
	}()
	return ret
}

func (flvWriter *FLVWriter) Write(p []byte) (err error) {
	err = nil
	if flvWriter.Closed {
		err = errors.New("flvwrite source closed")
		return
	}
	defer func() {
		if e := recover(); e != nil {
			errString := fmt.Sprintf("FLVWriter has already been closed:%v", e)
			err = errors.New(errString)
		}
	}()
	//fmt.Println("flvwriter队列长度：", len(flvWriter.packetQueue))
	flvWriter.packetQueue <- p

	return
}

func (flvWriter *FLVWriter) SendPacket() error {
	for {
		p, ok := <-flvWriter.packetQueue
		if ok {
			h := flvWriter.buf[:headerLen]

			preDataLen := len(p)

			if _, err := flvWriter.ctx.Write(p[:headerLen]); err != nil {
				return err
			}

			if _, err := flvWriter.ctx.Write(p[headerLen:]); err != nil {
				return err
			}

			utils.PutI32BE(h[:4], int32(preDataLen))
			if _, err := flvWriter.ctx.Write(h[:4]); err != nil {
				return err
			}
		} else {
			return errors.New("closed")
		}

	}

	return nil
}

func (flvWriter *FLVWriter) Wait() {
	select {
	case <-flvWriter.closedChan:
		return
	}
}

func (flvWriter *FLVWriter) Close() {
	log.Println("http flv closed")
	if !flvWriter.Closed {
		close(flvWriter.packetQueue)
		close(flvWriter.closedChan)
	}
	flvWriter.Closed = true
}

type Info struct {
	Uid string
	Key string
	Url string
}

func (flvWriter *FLVWriter) GetInfo() (ret *Info) {
	ret = &Info{
		Uid: flvWriter.Uid,
		Key: flvWriter.app + "/" + flvWriter.title,
		Url: flvWriter.url,
	}
	return ret
}
