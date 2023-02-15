package flv

type FlvPacket struct {
	Header  *FlvHeader
	Payload []*TagPacket
}

func NewFlvPacket() *FlvPacket {
	flv := &FlvPacket{
		Header:  NewFlvHeader(),
		Payload: make([]*TagPacket, 0),
	}
	return flv
}

// ==================================================
//					Flv Header
// ==================================================
type FlvHeader struct {
	tagF         byte   // "F" 0x46
	tagL         byte   // "L" 0x4c
	tagV         byte   // "V" 0x56
	version      byte   // 0x01
	flvType      byte   // 前五位保留为0, 当有音频时第六位置1,第七位保留为0, 当有视频时第八位置1
	headerLength uint32 // 头部长度 一般为 0x00, 0x00, 0x00, 0x09
}

func NewFlvHeader() *FlvHeader {
	return &FlvHeader{
		tagF:         0x46,
		tagL:         0x4c,
		tagV:         0x56,
		version:      0x01,
		flvType:      0x05,
		headerLength: uint32(9),
	}
}

func (flvHeader *FlvHeader) ToByte() (result []byte) {
	return []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}
}

// 修改 Flv Type
func (flvHeader *FlvHeader) FlvType(flvType byte) {
	// 检查 type 是否合法
	switch flvType {
	case FLV_TYPE_VIDEO, FLV_TYPE_AUDIO, FLV_TYPE_AUDIO_VIDEO:
		flvHeader.flvType = flvType
	default:
		// 输入不合法
	}
}

// flv 是否存在音频
// ** 当有音频时第六位置 == 1 **
func (flvHeader *FlvHeader) IsExitAudio() bool {
	return flvHeader.flvType == FLV_TYPE_AUDIO
}

// flv 是否存在视频
// ** 有视频时第八位置 == 1 **
func (flvHeader *FlvHeader) IsExitVideo() bool {
	return flvHeader.flvType == FLV_TYPE_VIDEO
}
