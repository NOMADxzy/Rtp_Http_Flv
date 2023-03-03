package h264

import (
	"bytes"
	"log"

	"Rtp_Http_Flv/container/h264"
)

var (
	StartCode2 = []byte{0x00, 0x00, 0x01}
	StartCode3 = []byte{0x00, 0x00, 0x00, 0x01}

	num = 0
)

const (
	NALU_MAX_SIZE = 1024 * 100
)

type Parser struct {
	// 解析完成待处理缓冲区
	NaluCache chan *h264.NaluPack
	// 解析对象
	Nalu *h264.NaluPack
	// 待解析 H.264 码流
	h264Buff chan []byte
}

func NewParser() *Parser {
	parser := &Parser{
		NaluCache: make(chan *h264.NaluPack, 10),
		Nalu:      h264.NewNaluPack(),
		h264Buff:  make(chan []byte, 64),
	}

	// 启动解析器
	go parser.parser()
	return parser
}

func (parser *Parser) WriteByte(payload []byte) {
	parser.h264Buff <- payload
}

func (parser *Parser) getAnnexbNalu(src []byte, startCode []byte) []byte {
	// 判断是否有起始码
	offset := isStartCode(src, startCode)

	if offset >= 0 {
		// 往 Nalu 添加数据
		if offset != 0 {
			parser.Nalu.Write(src[:offset])
		}

		if parser.Nalu.Length() > 0 {
			if num < 5 {
				log.Println(parser.Nalu.Data)
			}
			if len(startCode) == 3 {
				log.Println("001 -> ", parser.Nalu.Data)
			}
			parser.NaluCache <- parser.Nalu
			parser.Nalu = h264.NewNaluPack()
			num += 1
			log.Printf("解析第 %d 个 NALU \n", num)

		}

		offset += len(startCode)
		src = src[offset:]

		// 递归查找
		return parser.getAnnexbNalu(src, startCode)
	}

	return src
}

// 解析
func (parser *Parser) parser() {

	// 循环获取 H.264 码流
	// 从码流中搜索0x000001和0x00000001，分离出 NALU
	var (
		lastBuff []byte
	)

	for {
		select {
		case buff, ok := <-parser.h264Buff:
			if !ok {
				return
			}

			// 追加 buff
			lastBuff = append(lastBuff, buff...)
			lastBuff = parser.getAnnexbNalu(lastBuff, StartCode3)
			lastBuff = parser.getAnnexbNalu(lastBuff, StartCode2)

			if len(lastBuff) > 3 && parser.Nalu.Length() > 0 {
				parser.Nalu.Write(lastBuff[:len(lastBuff)-3])
			}
		}
	}
}

func isStartCode(src []byte, startCode []byte) int {
	if len(src) < len(startCode) {
		return -1
	}

	return bytes.Index(src, startCode)
}

func isStartCode3(src []byte) int {
	if len(src) < 4 {
		return -1
	}

	return bytes.Index(src, []byte{0x00, 0x00, 0x00, 0x01})
	//return src[0] == 0x00 &&
	//	src[1] == 0x00 &&
	//	src[2] == 0x00 &&
	//	src[3] == 0x01
}
