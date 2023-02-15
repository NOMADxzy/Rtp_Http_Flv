package ps

var PsStartCode = []byte{0x00, 0x00, 0x01}

// PS 包
type PsPack struct {
	Header      *PsHeader
	SystemTitle *SystemHeader
	PSM         *Psm
}

// PS 包头格式
type PsHeader struct {
	PackStartCode      []byte // 4 byte 包起始码字段, 0x000001BA 的位串，用来标志一个包的开始。
	SCRE               []byte // 6 byte 系统时钟参考字段
	MUXRate            []byte // 3 byte 节目复合速率字段
	PackStuffingLength int    // 3 bit 包填充长度字段
}

// 当且仅当pack是第一个数据包时才存在
type SystemHeader struct {
	PackStartCode []byte // 4 byte 包起始码字段, 0x000001BB 的位串，系统标题的开始。
	HeaderLength  byte   // 1 byte 头部长度
	// ... todo
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

type Psm struct {
	PacketStartCodePrefix     uint32 // 24 bit 分组起始码前缀字段。它和跟随其后的map_stream_id共同组成一个分组起始码以标志分组的开始
	MapStreamId               byte   // 4 bit 映射流标识字段 0xBC
	ProgramStreamMapLength    uint16 // 16 bit 节目流映射长度字段
	ProgramStreamInfoLength   uint16 // 16 bit 节目流信息长度字段
	ElementaryStreamMapLength uint16 // 16 bit 基本流映射长度字段. 说明了后面还有多少个个字节
	StreamType                byte   // 1 bit 表示流类型字段
	ElementaryStreamId        byte   // 1 bit 基本流标识字段
}

type Pes struct {
	PacketStartCodePrefix uint32 // 24 bit 分组起始码前缀字段。它和跟随其后的stream_id共同组成一个分组起始码以标志分组的开始
	StreamId              byte   // 4 bit 0x(C0~DF)指音频，0x(E0~EF)为视频
	PackLength            uint16 // 16 bit 指出了PES 分组中跟在该字段后的字节数目
}
