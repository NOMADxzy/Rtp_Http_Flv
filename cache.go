package main

import (
	"Rtp_Http_Flv/container/flv"
	"Rtp_Http_Flv/utils"
)

type SegmentPacket struct {
	full bool
	p    *flv.Packet
}

func (segmentPacket *SegmentPacket) Write(p *flv.Packet) {
	segmentPacket.p = p
	segmentPacket.full = true
}

func (segmentPacket *SegmentPacket) Send(w *FLVWriter) error {
	if !segmentPacket.full {
		return nil
	}

	// demux in hls will change p.Data, only send a copy here
	initialPacket := segmentPacket.p
	return w.Write(initialPacket.Data)
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

func (cache *SegmentCache) Write(p *flv.Packet) {
	err := flv.DemuxHeader(p)
	utils.CheckError(err)

	if p.IsAudio { //是音频初始段
		if ah, ok := p.Header.(flv.AudioPacketHeader); ok {
			if ah.SoundFormat() == flv.SOUND_AAC && ah.AACPacketType() == flv.AAC_SEQHDR {
				cache.audioSeq.Write(p)
			}
		}

	} else if p.IsVideo {
		if vh, ok := p.Header.(flv.VideoPacketHeader); ok {
			if vh.IsSeq() {
				cache.videoSeq.Write(p)
			}
		}
	}

	if cache.videoSeq.full && cache.audioSeq.full {
		cache.full = true
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
