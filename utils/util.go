package utils

import (
	"Rtp_Http_Flv/configure"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func Get(url string) map[string]interface{} {

	// 超时时间：5秒
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		CheckError(err)
	}(resp.Body)
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
	err = json.Unmarshal(result.Bytes(), &res)
	CheckError(err)
	if res["data"].(map[string]interface{})["publishers"] == nil {
		return nil
	}
	return res
}

type Publisher struct {
	Key               string
	url               string
	StartTime         int64
	Ssrc              uint32
	id                uint32
	video_total_bytes uint64
	video_speed       uint64
	audio_total_bytes uint64
	audio_speed       uint64
}

func UpdatePublishers() map[uint32]*Publisher {
	newPublishers := make(map[uint32]*Publisher) //清空map

	res := Get("http://" + configure.CLOUD_HOST + configure.API_ADDR + "/stat/livestat")
	if res == nil {
		return nil
	}
	pubs := res["data"].(map[string]interface{})["publishers"].([]interface{})

	for _, pub := range pubs {
		p := pub.(map[string]interface{})

		newPublishers[uint32(p["ssrc"].(float64))] = &Publisher{
			Key:               p["key"].(string),
			url:               p["url"].(string),
			StartTime:         int64(p["start_time"].(float64)),
			Ssrc:              uint32(p["ssrc"].(float64)),
			id:                uint32(p["stream_id"].(float64)),
			video_total_bytes: uint64(p["video_total_bytes"].(float64)),
			video_speed:       uint64(p["video_speed"].(float64)),
			audio_total_bytes: uint64(p["audio_total_bytes"].(float64)),
			audio_speed:       uint64(p["audio_speed"].(float64)),
		}
	}
	return newPublishers
}
func CreateFlvFile(name string) *File {
	flvFile, err := CreateFile(configure.RECORD_DIR + "/" + name + ".flv")
	if err != nil {
		fmt.Println("Create FLV dump file error:", err)
		return nil
	}
	return flvFile
}

func IsTagHead(payload []byte) bool {
	if len(payload) < 11 {
		return false
	}
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

func IsPacketLoss() bool {
	r := rand.Intn(10000)
	if float64(r)/10000.0 >= configure.PACKET_LOSS_RATE {
		return false
	} else {
		return true
	}
}

func PutI32BE(b []byte, v int32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}
func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

func FirstBeforeSecond(seq1 uint16, seq2 uint16) bool {
	if seq1 < seq2 {
		return seq2-seq1 < uint16(60000)
	} else {
		return seq1-seq2 > uint16(60000)
	}
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
