package rtp

import (
	"encoding/binary"

	"rtp_http_flv/container/rtp"
)

type Parser struct {
}

func (rtpParser *Parser) Parse(rtpBytes []byte) *rtp.RtpPack {
	if len(rtpBytes) < rtp.RTP_FIXED_HEADER_LENGTH {
		return nil
	}

	firstByte := rtpBytes[0]
	secondByte := rtpBytes[1]

	pack := &rtp.RtpPack{
		Version:   firstByte >> 6,
		Padding:   (firstByte>>5)&1 == 1,
		Extension: (firstByte>>4)&1 == 1,
		CSRCCnt:   firstByte & 0x0f,

		Marker:         secondByte >> 7,
		PayloadType:    secondByte & 0x7f,
		SequenceNumber: binary.BigEndian.Uint16(rtpBytes[2:]),
		Timestamp:      binary.BigEndian.Uint32(rtpBytes[4:]),
		SSRC:           binary.BigEndian.Uint32(rtpBytes[8:]),
	}
	offset := rtp.RTP_FIXED_HEADER_LENGTH
	end := len(rtpBytes)
	// 每个 CSRC 标识符占32位
	if end-offset >= 4*int(pack.CSRCCnt) {
		offset += 4 * int(pack.CSRCCnt)
	}

	// Extension == 1, 则在RTP报头后跟有一个扩展报头
	if pack.Extension && end-offset >= 4 {
		extLen := 4 * int(binary.BigEndian.Uint16(rtpBytes[offset+2:]))
		offset += 4
		if end-offset >= extLen {
			offset += extLen
		}
	}

	// 如果P=1，则在该报文的尾部填充一个或多个额外的八位组
	if pack.Padding && end-offset > 0 {
		paddingLen := int(rtpBytes[end-1])
		if end-offset >= paddingLen {
			end -= paddingLen
		}
	}
	pack.Payload = rtpBytes[offset:end]
	pack.PayloadOffset = offset
	if end-offset < 1 {
		return nil
	}

	// 解析

	return pack
}
