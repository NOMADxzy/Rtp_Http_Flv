package utils

import (
	"Rtp_Http_Flv/configure"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

func UintToBytes(val uint, index int) []byte {
	b := make([]byte, index)
	for i := 0; i < index; i++ {
		b[i] = byte(val >> (8 * (index - i - 1)))
	}
	return b
}

func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)

	return bytes
}

// 返回 uint32
func BytesToUint32(val []byte) uint32 {
	return uint32(val[0]<<24) + uint32(val[1]<<16) + uint32(val[2]<<8) + uint32(val[3])
}

func AmfStringToBytes(b *bytes.Buffer, val string) {
	b.Write(UintToBytes(uint(len(val)), 2))
	b.Write([]byte(val))
}

func AmfDoubleToBytes(b *bytes.Buffer, val float64) {
	b.WriteByte(0x00)
	b.Write(Float64ToByte(val))
}

func Get(url string) map[string]interface{} {

	// 超时时间：5秒
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}

	var res map[string]interface{}
	json.Unmarshal(result.Bytes(), &res)
	if res["data"].(map[string]interface{})["publishers"] == nil {
		return nil
	}
	return res
}

type Publisher struct {
	Key               string
	url               string
	Ssrc              uint32
	id                uint32
	video_total_bytes uint64
	video_speed       uint64
	audio_total_bytes uint64
	audio_speed       uint64
}

func UpdatePublishers(publishers map[uint32]*Publisher) {

	res := Get(configure.API_URL + "/stat/livestat")
	if res == nil {
		return
	}
	pubs := res["data"].(map[string]interface{})["publishers"].([]interface{})

	for _, pub := range pubs {
		p := pub.(map[string]interface{})

		publishers[uint32(p["ssrc"].(float64))] = &Publisher{
			Key:               p["key"].(string),
			url:               p["url"].(string),
			Ssrc:              uint32(p["ssrc"].(float64)),
			id:                uint32(p["stream_id"].(float64)),
			video_total_bytes: uint64(p["video_total_bytes"].(float64)),
			video_speed:       uint64(p["video_speed"].(float64)),
			audio_total_bytes: uint64(p["audio_total_bytes"].(float64)),
			audio_speed:       uint64(p["audio_speed"].(float64)),
		}
	}
}
func CreateFlvFile(name string) *File {
	flvFile, err := CreateFile("./" + name + ".flv")
	if err != nil {
		fmt.Println("Create FLV dump file error:", err)
		return nil
	}
	return flvFile
}

func IsTagHead(payload []byte) bool {
	if payload[0] == byte(8) || payload[0] == byte(9) {
		if payload[8] == byte(0) && payload[9] == byte(0) && payload[10] == byte(0) {
			tmpBuf := make([]byte, 4)
			copy(tmpBuf[1:], payload[1:4])
			TagSize := int(uint32(tmpBuf[1])<<16 | uint32(tmpBuf[2])<<8 | uint32(tmpBuf[3]) + uint32(11))
			return TagSize == len(payload)
		}
	}
	return false
}

func PutI32BE(b []byte, v int32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}
