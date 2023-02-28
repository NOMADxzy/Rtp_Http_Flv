package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go-mpu/configure"
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

var VideoInitializationSegment = []byte{
	9, 0, 0, 56, 0, 0, 0, 0, 0, 0,
	0, 23, 0, 0, 0, 0, 1, 100, 0, 40,
	255, 225, 0, 30, 103, 100, 0, 40, 172, 217,
	64, 120, 2, 39, 229, 192, 90, 128, 128, 128,
	160, 0, 0, 3, 0, 32, 0, 0, 7, 129,
	227, 6, 50, 192, 1, 0, 6, 104, 235, 227,
	203, 34, 192, 253, 248, 248, 0}
var AudioInitializationSegment = []byte{
	8, 0, 0, 7, 0, 0, 0, 0, 0, 0,
	0, 175, 0, 18, 16, 86, 229, 0,
}
