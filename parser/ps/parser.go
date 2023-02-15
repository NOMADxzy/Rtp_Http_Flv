package ps

import (
	"fmt"
	"log"

	"go-mpu/container/ps"
)

type Parser struct {
}

// PS 包头格式
type PsHeader struct {
	PackStartCode      []byte // 4 byte 包起始码字段
	SCRE               []byte // 6 byte 系统时钟参考字段
	MUXRate            []byte // 3 byte 节目复合速率字段
	PackStuffingLength int    // 3 bit 包填充长度字段
}

// PS map格式
//
// StreamType 参考:
// MPEG-4  视频流： 0x10
// H.264   视频流： 0x1B
// SVAC    视频流： 0x80
// G.711   音频流： 0x90
// G.722.1 音频流： 0x92
// G.723.1 音频流： 0x93
// G.729   音频流： 0x99
// SVAC    音频流： 0x9B

type psMap struct {
	PacketStartCodePrefix     int // 24 bit 分组起始码前缀字段。它和跟随其后的map_stream_id共同组成一个分组起始码以标志分组的开始
	MapStreamId               int // 1 byte 映射流标识字段 0xBC
	ProgramStreamMapLength    int // 1 byte 节目流映射长度字段
	ProgramStreamInfoLength   int // 2 byte 节目流信息长度字段
	ElementaryStreamMapLength int // 2 byte 基本流映射长度字段. 说明了后面还有多少个个字节
	StreamType                int // 1 byte 表示流类型字段
	ElementaryStreamId        int // 1 byte 基本流标识字段
}

func NewParser() *Parser {
	return &Parser{}
}

func isStartCode(src []byte) bool {
	if len(src) < 3 {
		return false
	}
	return src[0] == 0x00 &&
		src[1] == 0x00 &&
		src[2] == 0x01
}

func isPsHeader(src []byte) bool {
	if len(src) < 4 {
		return false
	}
	return src[0] == 0x00 &&
		src[1] == 0x00 &&
		src[2] == 0x01 &&
		src[3] == 0xba
}

func isPsmHeader(src []byte) bool {
	if len(src) < 4 {
		return false
	}
	return src[0] == 0x00 &&
		src[1] == 0x00 &&
		src[2] == 0x01 &&
		(src[3] == 0xbb || src[3] == 0xbc)
}

// 获取 ps 荷载的真实数据
func GetPsPayload(psPack []byte) []byte {
	// 解析 ps header
	if len(psPack) < 4 {
		return psPack
	}

	// 解析位置
	offset := 0
	for len(psPack) > offset {
		// 是否有起始码 {0x00 0x00 0x01}
		if !isStartCode(psPack[offset:]) {
			return psPack[offset:]
		}

		offset += 3

		// 判断 type
		streamType := psPack[offset]

		if streamType == 0xba {
			psHeader := &PsHeader{
				PackStartCode:      psPack[(offset - 3):(offset + 1)],
				SCRE:               psPack[(offset + 1):(offset + 7)],
				MUXRate:            psPack[(offset + 7):(offset + 10)],
				PackStuffingLength: int(psPack[(offset+10)] & 0x03),
			}

			offset += 11
			// 遇到扩展字段，跳过不解析, 扩展字段 占 1 byte
			if psHeader.PackStuffingLength > 0 {
				offset += psHeader.PackStuffingLength
			}

			//fmt.Println(offset)

		} else if streamType == 0xbb { // 解析 PS System Header 和 Program Stream Map（PSM）节目映射流 （只有第一个包才有）
			offset += 1
			fmt.Println("解析第一个 RTP 包 -> PS System Header")
			// System Header
			headerLength := int(psPack[offset])<<8 | int(psPack[offset+1])
			if headerLength > 0 {
				offset += headerLength
			}
			offset += 2

		} else if streamType == 0xbc {
			offset += 1
			fmt.Println("解析第一个 RTP 包 -> Program Stream Map（PSM）")
			psm := &ps.Psm{
				PacketStartCodePrefix: 0x01,
				MapStreamId:           streamType,
			}
			// System Header
			psm.ProgramStreamMapLength = uint16(psPack[offset]<<8 | psPack[offset+1])
			if psm.ProgramStreamMapLength > 0 {
				// todo 不解析 下面的数据的话 直接 跳过这个长度就可以了
				//offset += int(psm.ProgramStreamMapLength)
			}

			// 跳过两个无用字节, 获取接下来的 节目流信息长度字段
			offset += 4
			psm.ProgramStreamInfoLength = uint16(psPack[offset]<<8 | psPack[offset+1])
			if psm.ProgramStreamInfoLength > 0 {
				// 跳过 描述符的总长度
				offset += int(psm.ProgramStreamInfoLength)
			}
			offset += 2
			// 获取 基本流映射长度字段，根据该字段循环
			psm.ElementaryStreamMapLength = uint16(psPack[offset]<<8 | psPack[offset+1])
			offset += 2
			for i := 0; i < int(psm.ElementaryStreamMapLength); i++ {
				// 流类型字段
				streamType := psPack[offset+i]
				// 流 ID
				streamId := psPack[offset+i+1]
				// 跳过描述字节
				descLen := int(psPack[offset]<<8 | psPack[offset+1])
				i += descLen
				log.Printf("PSM -> streamType(%v) + streamId(%v) \n", streamType, streamId)
			}
			offset += int(psm.ElementaryStreamMapLength)

			// 最后 4 byte 是 CRC_32
			offset += 6
		} else if streamType == 0xe0 {
			//} else {
			// 判断是否有 Pts 和 Dts
			// 跳过 四个 字节 获取 pes 头部长度
			offset += 5
			offset += int(psPack[offset]) + 1
			fmt.Println("解析一个 PES 包 -> PES")
			//fmt.Println("解析一个 PES 包 -> PES -> ", len(psPack[offset:]))
			// H.264 Data
			return psPack[offset:]
		}

	}

	return make([]byte, 0)
}
