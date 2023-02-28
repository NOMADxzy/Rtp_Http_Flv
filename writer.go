package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gwuhaolin/livego/av"
	"github.com/gwuhaolin/livego/utils/pio"
)

const (
	headerLen   = 11
	maxQueueNum = 10240
)

type FLVWriter struct {
	Uid string
	av.RWBaser
	app, title, url string
	buf             []byte
	closed          bool
	closedChan      chan struct{}
	ctx             http.ResponseWriter
	packetQueue     chan []byte
	init            bool
}

func NewFLVWriter(app, title, url string, ctx http.ResponseWriter) *FLVWriter {
	ret := &FLVWriter{
		Uid:         "测试id-fjaefiagrklvg",
		app:         app,
		title:       title,
		url:         url,
		ctx:         ctx,
		RWBaser:     av.NewRWBaser(time.Second * 10),
		closedChan:  make(chan struct{}),
		buf:         make([]byte, headerLen),
		packetQueue: make(chan []byte, maxQueueNum),
	}

	ret.ctx.Write([]byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09})
	pio.PutI32BE(ret.buf[:4], 0)
	ret.ctx.Write(ret.buf[:4])
	go func() {

		err := ret.SendPacket()
		if err != nil {
			log.Println("SendPacket error:", err)
			ret.closed = true
		}
	}()
	return ret
}

func (flvWriter *FLVWriter) Write(p []byte) (err error) {
	err = nil
	if flvWriter.closed {
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

			pio.PutI32BE(h[:4], int32(preDataLen))
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

func (flvWriter *FLVWriter) Close(error) {
	log.Println("http flv closed")
	if !flvWriter.closed {
		close(flvWriter.packetQueue)
		close(flvWriter.closedChan)
	}
	flvWriter.closed = true
}

func (flvWriter *FLVWriter) Info() (ret av.Info) {
	ret.UID = flvWriter.Uid
	ret.URL = flvWriter.url
	ret.Key = flvWriter.app + "/" + flvWriter.title
	ret.Inter = true
	return
}
