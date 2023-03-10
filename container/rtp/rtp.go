package rtp

import (
	"fmt"
	//"Rtp_Http_Flv/parser/ps"
)

type RtpPack struct {
	Version        byte   // 2 bit, 版本号 2
	Padding        bool   // 1 bit, 填充标志, 如果P=1，则在该报文的尾部填充一个或多个额外的八位组
	Extension      bool   // 1 bit, 扩展标志, 如果X=1，则在 RTP报头 后跟有一个扩展报头
	CSRCCnt        byte   // 4 bit, CSRC计数器, 指示CSRC 标识符的个数
	Marker         byte   // 1 bit, 标记, 对于视频，标记一帧的结束；对于音频，标记会话的开始
	PayloadType    byte   // 7 bit, 有效荷载类型, (96 - PS)，(97 - MPEG-4)，(98 - H264)
	SequenceNumber uint16 // 2 byte, 序列号
	Timestamp      uint32 // 4 byte, 时间戳, 必须使用90 kHz 时钟频率
	SSRC           uint32 // 4 byte, 同步信源(SSRC)标识符
	Payload        []byte
	PayloadOffset  int
}

func (rtp *RtpPack) RealData() []byte {
	buff := make([]byte, 0)

	switch rtp.PayloadType {
	case 96:
		// PS 格式
		//buff = ps.GetPsPayload(rtp.Payload)
	case 97:
		// todo 未实现：MPEG-4 格式
	case 98:
		// todo 未实现：H264 格式
	default:
		return buff
	}

	return buff
}

func (rtp *RtpPack) String() string {
	return fmt.Sprintf("Version: %v, Padding: %v, Marker: %v, PayloadType: %v, SequenceNumber: %v, Timestamp: %v, PayloadOffset: %v",
		rtp.Version,
		rtp.Padding,
		rtp.Marker,
		rtp.PayloadType,
		rtp.SequenceNumber,
		rtp.Timestamp,
		rtp.PayloadOffset,
	)
}
