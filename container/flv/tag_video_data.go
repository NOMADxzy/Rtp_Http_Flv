package flv

import (
	"bytes"
	"encoding/binary"

	"rtp_http_flv/container/h264"
	"rtp_http_flv/utils"
)

// ==========================================
//
//	Video Tag Data
//
// ==========================================
// codecID = 7 , AvcVideoPacket
type AvcVideoPacket struct {
	/*
		1: keyframe (for AVC, a seekable frame)
		2: inter frame (for AVC, a non- seekable frame)
		3: disposable inter frame (H.263 only)
		4: generated keyframe (reserved for server use only)
		5: video info/command frame
	*/
	FrameType uint8
	/*
		1: JPEG (currently unused)
		2: Sorenson H.263
		3: Screen video
		4: On2 VP6
		5: On2 VP6 with alpha channel
		6: Screen video version 2
		7: AVC
	*/
	codecId uint8
	/*
		0: AVC sequence header
		1: AVC NALU
		2: AVC end of sequence (lower level NALU sequence ender is not required or supported)
	*/
	AvcPacketType uint8

	/*
		if AvcPacketType == 0
			CompositionTime = 0
	*/
	compositionTime uint32

	/*
		if AvcPacketType == 0
			Data = AVCDecoderConfigurationRecord
		if AvcPacketType == 1
					Data = NaluPack
		if AvcPacketType == 2
					Data = Empty
	*/
	Data UnitData
}

func NewAvcVideoPacket() *AvcVideoPacket {
	return &AvcVideoPacket{
		FrameType:       0x01,
		codecId:         0x07,
		AvcPacketType:   0x01,
		compositionTime: 0,
	}
}

func (avc *AvcVideoPacket) SetData(data UnitData) {
	avc.Data = data
}

func (avc *AvcVideoPacket) IsKeyFrame() bool {
	return avc.FrameType == 1
}

func (avc *AvcVideoPacket) IsSeq() bool {
	return avc.IsKeyFrame() && avc.AvcPacketType == 0
}

func (avc *AvcVideoPacket) CodecId() uint8 {
	return 0x07
}

func (avc *AvcVideoPacket) SetCompositionTime(time uint32) {
	avc.compositionTime = time
}

func (avc *AvcVideoPacket) CompositionTime() uint32 {
	if avc.AvcPacketType == 1 {
		return avc.compositionTime
	}
	return 0
}

func (avc *AvcVideoPacket) ToTagBuffer() *bytes.Buffer {
	var buff bytes.Buffer
	buff.Write([]byte{
		avc.FrameType<<4 + avc.codecId,
		avc.AvcPacketType,
	})
	// compositionTime
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, avc.compositionTime)
	buff.Write(b[1:])

	buff.Write(avc.Data.ToAvcVideoBuffer().Bytes())
	return &buff
}

type UnitData interface {
	Unit() []byte
	ToAvcVideoBuffer() *bytes.Buffer
}

// AVCPacketType = 0，那么Data就是 AVCDecoderConfigurationRecord 格式
type AVCDecoderConfigurationRecord struct {
	ConfigVersion        byte //8bits，版本号, 0x01
	AvcProfileIndication byte //8bits，sps[1]
	ProfileCompatility   byte //8bits，sps[2]
	AvcLevelIndication   byte //8bits，sps[3]
	// NALUSize的长度，计算方法为：1 + (lengthSizeMinusOne & 3)=4
	reserved byte //6bits，111111
	NaluLen  byte //2bits，NALUSize的长度
	// 低五位为SPS的个数，计算方法为：numOfSequenceParameterSets & 0x1F=1
	reserved1 byte //3bits, 111
	SpsNum    byte //5bits
	SpsSize   uint
	Sps       []byte
	PpsNum    byte //8bits
	PpsSize   uint
	Pps       []byte
}

func NewAVCDecoderConfigurationRecord() *AVCDecoderConfigurationRecord {
	return &AVCDecoderConfigurationRecord{
		ConfigVersion:        0x01,
		AvcProfileIndication: 0,
		ProfileCompatility:   0,
		AvcLevelIndication:   0,
		reserved:             0x3f,
		NaluLen:              4,
		reserved1:            0x07,
		SpsNum:               0x01,
		SpsSize:              0,
		Sps:                  make([]byte, 0),
		PpsNum:               0x01,
		PpsSize:              0,
		Pps:                  make([]byte, 0),
	}
}

func (pack *AVCDecoderConfigurationRecord) Unit() []byte {
	return pack.ToAvcVideoBuffer().Bytes()
}

func (pack *AVCDecoderConfigurationRecord) ToAvcVideoBuffer() *bytes.Buffer {
	var buff bytes.Buffer
	buff.Write([]byte{
		pack.ConfigVersion,
		pack.AvcProfileIndication,
		pack.ProfileCompatility,
		pack.AvcLevelIndication,
		(pack.NaluLen - 1) | 0xfc,
		pack.SpsNum | 0xe0,
	})

	// SpsSize 的长度
	buff.Write(utils.UintToBytes(pack.SpsSize, 2))
	// Sps
	buff.Write(pack.Sps)
	// PpsNum
	buff.WriteByte(pack.PpsNum)
	// PpsSize 的长度
	buff.Write(utils.UintToBytes(pack.PpsSize, 2))
	// Pps
	buff.Write(pack.Pps)

	return &buff
}

// AVCPacketType = 1，那么Data就是 Nalu Len + Nalu = NaluPack
// goto h264.NaluPack

type NaluUnitData struct {
	slice []*h264.NaluPack
}

func NewNaluUnitData() *NaluUnitData {
	return &NaluUnitData{slice: make([]*h264.NaluPack, 0)}
}

func (naluUnitData *NaluUnitData) SetData(nalu *h264.NaluPack) {
	naluUnitData.slice = append(naluUnitData.slice, nalu)
}

func (naluUnitData *NaluUnitData) Unit() []byte {
	return naluUnitData.ToAvcVideoBuffer().Bytes()
}

func (naluUnitData *NaluUnitData) ToAvcVideoBuffer() *bytes.Buffer {
	var buff bytes.Buffer
	for _, nalu := range naluUnitData.slice {
		naluLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(naluLenBytes, nalu.Length())
		buff.Write(naluLenBytes)
		buff.Write(nalu.Data)
	}
	return &buff
}
