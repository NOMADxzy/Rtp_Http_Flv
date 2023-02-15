package h264

const (
	BufferSize uint16 = 65535 - 20 - 8

	NALU_TYPE_SLICE    byte = 0x01
	NALU_TYPE_DPA           = 0x02
	NALU_TYPE_DPB           = 0x03
	NALU_TYPE_DPC           = 0x04
	NALU_TYPE_IDR           = 0x05
	NALU_TYPE_SEI           = 0x06
	NALU_TYPE_SPS           = 0x07
	NALU_TYPE_PPS           = 0x08
	NALU_TYPE_AUD           = 0x09
	NALU_TYPE_EOSEQ         = 0x10
	NALU_TYPE_EOSTREAM      = 0x11
	NALU_TYPE_FILL          = 0x12
)

var (
	StartCode = []byte{0x00, 0x00, 0x00, 0x01}
)

// NALU 基本单元包
type NaluPack struct {
	length uint32
	Data   []byte
}

func NewNaluPack() *NaluPack {
	return &NaluPack{
		length: 0,
		Data:   make([]byte, 0),
	}
}

func (pack *NaluPack) IsSps() bool {
	if len(pack.Data) <= 0 {
		return false
	}
	return pack.Data[0]&0x1f == NALU_TYPE_SPS
}

func (pack *NaluPack) IsPps() bool {
	if len(pack.Data) <= 0 {
		return false
	}
	return pack.Data[0]&0x1f == NALU_TYPE_PPS
}

func (pack *NaluPack) IsKey() bool {
	if len(pack.Data) <= 0 {
		return false
	}
	return pack.Data[0]&0x1f == NALU_TYPE_IDR
}

func (pack *NaluPack) IsSet() bool {
	if len(pack.Data) <= 0 {
		return false
	}
	return pack.Data[0]&0x1f == NALU_TYPE_SEI
}

func (pack *NaluPack) Write(val []byte) {
	pack.Data = append(pack.Data, val...)
}

func (pack *NaluPack) Length() uint32 {
	return uint32(len(pack.Data))
}
