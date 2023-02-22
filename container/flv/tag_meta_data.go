package flv

import (
	"bytes"

	"rtp_http_flv/utils"
)

// ==========================================
//
//	Script Tag Data
//
// ==========================================
type MetaPacket struct {
	Buff bytes.Buffer
}

func NewMetaTagData() *MetaPacket {
	return &MetaPacket{}
}

// 初始化
func (payload *MetaPacket) Init() {
	// ============= 第一个 AMF
	// 1. 第一个AFM包，类型为0x02，表示字符串
	payload.Buff.WriteByte(0x02)
	// 2. 写入字符串的长度（UI16），然后再写入字符串
	utils.AmfStringToBytes(&payload.Buff, "onMetaData")

	// ============= 第二个 AMF
	// 第二个AFM包，类型为0x08，表示数组
	payload.Buff.WriteByte(0x08)
	// 3. 写入数组长度，7个数组项
	payload.Buff.Write(utils.UintToBytes(uint(AMF_VIDEO_ECMA_ARRAY_LENGTH), 4))

	// -----duration，视频的时长
	utils.AmfStringToBytes(&payload.Buff, "duration")
	utils.AmfDoubleToBytes(&payload.Buff, 0) // 写0只是为了占位

	// -----width, 宽度
	utils.AmfStringToBytes(&payload.Buff, "width")
	utils.AmfDoubleToBytes(&payload.Buff, 320)

	// -----height，高度
	utils.AmfStringToBytes(&payload.Buff, "height")
	utils.AmfDoubleToBytes(&payload.Buff, 240)

	// -----videodatarate，视频码率
	utils.AmfStringToBytes(&payload.Buff, "videodatarate")
	utils.AmfDoubleToBytes(&payload.Buff, 0) // 占位

	// -----framerate，视频帧率
	utils.AmfStringToBytes(&payload.Buff, "framerate")
	utils.AmfDoubleToBytes(&payload.Buff, 15)

	// -----videocodecid，视频编码方式
	utils.AmfStringToBytes(&payload.Buff, "videocodecid")
	utils.AmfDoubleToBytes(&payload.Buff, FLV_CODECID_H264)

	// -----filesize，文件大小
	utils.AmfStringToBytes(&payload.Buff, "filesize")
	utils.AmfDoubleToBytes(&payload.Buff, 0) // 占位

	// -----END_OF_OBJECT，009
	utils.AmfStringToBytes(&payload.Buff, "")
	payload.Buff.Write(utils.UintToBytes(uint(AMF_END_OF_OBJECT), 2))
}

func (payload *MetaPacket) ToTagBuffer() *bytes.Buffer {
	return &payload.Buff
}
