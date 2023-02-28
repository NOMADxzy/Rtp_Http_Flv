package main

type SegmentPacket struct {
	full bool
	p    []byte
}

func (segmentPacket *SegmentPacket) Write(p []byte) {
	segmentPacket.p = p
	segmentPacket.full = true
}

func (segmentPacket *SegmentPacket) Send(w *FLVWriter) error {
	if !segmentPacket.full {
		return nil
	}

	// demux in hls will change p.Data, only send a copy here
	newPacket := segmentPacket.p
	return w.Write(newPacket)
}

type SegmentCache struct {
	videoSeq *SegmentPacket
	audioSeq *SegmentPacket
	metadata *SegmentPacket
	full     bool
}

func NewCache() *SegmentCache {
	return &SegmentCache{
		videoSeq: &SegmentPacket{},
		audioSeq: &SegmentPacket{},
		metadata: &SegmentPacket{},
	}
}

func (cache *SegmentCache) Write(p []byte) {
	if p[0] == byte(8) {
		cache.audioSeq.Write(p)
		if cache.videoSeq.full {
			cache.full = true
		}

	} else if p[0] == byte(9) {
		cache.videoSeq.Write(p)
		if cache.audioSeq.full {
			cache.full = true
		}
	}
}

func (cache *SegmentCache) SendInitialSegment(w *FLVWriter) error {
	if err := cache.metadata.Send(w); err != nil {
		return err
	}

	if err := cache.videoSeq.Send(w); err != nil {
		return err
	}

	if err := cache.audioSeq.Send(w); err != nil {
		return err
	}

	return nil
}
