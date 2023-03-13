package cache

type FlvRecord struct {
	flvTag         []byte //记录当前flvTag写入的字节情况
	TagSize        int
	pos            int  //写入flvTag的位置
	jumpToNextHead bool //发生丢包后跳到下一个tag头开始解析
}

func NewFlvRecord() *FlvRecord {
	return &FlvRecord{
		nil, 0, 0, false,
	}
}

func (flvRecord *FlvRecord) Reset() {
	flvRecord.flvTag = nil
	flvRecord.TagSize = 0
	flvRecord.pos = 0
}
