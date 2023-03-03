package flv

import (
	"bytes"
	"encoding/binary"

	"Rtp_Http_Flv/utils"
)

type TagPacket struct {
	TagType     TagPackType // Tag Type , 8 : audio , 9 : video , 18 : script data
	TagDataSize uint32      // 3 字节表示 Tag Data 大小
	Ts          uint32      // 3 字节表示 Tag 时间戳
	TsEx        byte        // 时间戳扩展位
	StreamId    uint32      // 一直为0

	Payload TagData
}

func NewTagPacket(tagType TagPackType) *TagPacket {
	tagPack := &TagPacket{
		TagType:     tagType,
		TagDataSize: 0,
		Ts:          0,
		TsEx:        0,
		StreamId:    0,
	}

	switch tagType {
	case TAG_TYPE_AUDIO:
		//
	case TAG_TYPE_VIDEO:
		tagPack.Payload = NewAvcVideoPacket()
	case TAG_TYPE_SCRIPT:
		tagPack.Payload = NewMetaTagData()
	}

	return tagPack
}

func (packet *TagPacket) IsAudio() bool {
	return packet.TagType == TAG_TYPE_AUDIO
}

func (packet *TagPacket) IsVideo() bool {
	return packet.TagType == TAG_TYPE_VIDEO
}

func (packet *TagPacket) IsMetadata() bool {
	return packet.TagType == TAG_TYPE_SCRIPT
}

func (packet *TagPacket) SetTagDataSize(val uint32) {
	packet.TagDataSize = val
}

func (packet *TagPacket) SetPayload(data TagData) {
	packet.SetTagDataSize(uint32(data.ToTagBuffer().Len()))
	packet.Payload = data
}

func (packet *TagPacket) Bytes() []byte {
	buff := make([]byte, 0)
	// header
	buff = append(buff, byte(packet.TagType))
	buff = append(buff, utils.UintToBytes(uint(packet.Payload.ToTagBuffer().Len()), 3)...)
	buff = append(buff, utils.UintToBytes(uint(packet.Ts), 3)...)
	buff = append(buff, packet.TsEx)
	buff = append(buff, utils.UintToBytes(uint(packet.StreamId), 3)...)
	// payload
	buff = append(buff, packet.Payload.ToTagBuffer().Bytes()...)
	// tag size
	tagSize := uint32(len(buff))
	tagSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(tagSizeBytes, tagSize)
	buff = append(buff, tagSizeBytes...)

	return buff
}

// Timestamp + TimestampExtended
// 组成了这个TAG包数据的PTS信息，真正数据的PTS = Timestamp | TimestampExtended << 24
func (packet *TagPacket) Pts() uint32 {
	return packet.Ts | uint32(packet.TsEx<<24)
}

// ===============================================
//					Tag Data
// ===============================================

type TagData interface {
	ToTagBuffer() *bytes.Buffer
}

// Video Tag Data
type VideoTagData interface {
	TagData
	IsKeyFrame() bool
	IsSeq() bool
	CodecId() uint8
	CompositionTime() uint32
}

// Audio Tag Data
type AudioTagData interface {
	TagData
	SoundFormat() uint8
	AACPacketType() uint8
}

// Script Tag Data
type MetaTagData interface {
	TagData
	Init()
}
