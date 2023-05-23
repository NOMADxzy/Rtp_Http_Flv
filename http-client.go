package main

import (
	"bytes"
	"encoding/json"

	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/NOMADxzy/livego/configure"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	AUDIO_TAG       = byte(0x08)
	VIDEO_TAG       = byte(0x09)
	SCRIPT_DATA_TAG = byte(0x12)
	DURATION_OFFSET = 53
	HEADER_LEN      = 13
	KEY_FRAME       = byte(0x17)
)

type TagHeader struct {
	TagType   byte
	DataSize  uint32
	Timestamp uint32
}

func ReadTag(reader io.ReadCloser) (header *TagHeader, data []byte, err error) {
	tmpBuf := make([]byte, 4)
	header = &TagHeader{}
	// Read tag header
	if _, err = io.ReadFull(reader, tmpBuf[3:]); err != nil {
		return
	}
	header.TagType = tmpBuf[3]

	// Read tag size
	if _, err = io.ReadFull(reader, tmpBuf[1:]); err != nil {
		return
	}
	header.DataSize = uint32(tmpBuf[1])<<16 | uint32(tmpBuf[2])<<8 | uint32(tmpBuf[3])

	// Read timestamp
	if _, err = io.ReadFull(reader, tmpBuf); err != nil {
		return
	}
	header.Timestamp = uint32(tmpBuf[3])<<24 + uint32(tmpBuf[0])<<16 + uint32(tmpBuf[1])<<8 + uint32(tmpBuf[2])

	// Read stream ID
	if _, err = io.ReadFull(reader, tmpBuf[1:]); err != nil {
		return
	}

	// Read data
	data = make([]byte, header.DataSize)
	if _, err = io.ReadFull(reader, data); err != nil {
		return
	}

	// Read previous tag size
	if _, err = io.ReadFull(reader, tmpBuf); err != nil {
		return
	}

	return
}

func httpClient(reqUrl string, timeout int) (client *http.Client, err error) {

	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("")
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Dial: (&net.Dialer{
				Timeout: time.Duration(timeout) * time.Second,
			}).Dial,
			ResponseHeaderTimeout: time.Second * time.Duration(timeout),
			DisableKeepAlives:     true,
		},
	}

	return
}

func getDelay(header *TagHeader, startTime int64) int {
	now := time.Now().UnixMilli()
	if delay := int(now - (startTime + int64(header.Timestamp))); delay < 0 {
		return 0
	} else {
		return delay
	}
}

func httpFlv(reqUrl string, id int, interval int) {
	configure.Init()
	publishers := UpdatePublishers()
	startTime := publishers[1020304].StartTime // 测试时只有第一个流
	TotalBytes := 0

	client, err2 := httpClient(reqUrl, 10)

	if err2 != nil {
		fmt.Println("client not sure")
	}
	request, _ := http.NewRequest("GET", reqUrl, nil)
	request.Header.Add("User-Agent", "curl/7.19.7 (x86_64-redhat-linux-gnu) libcurl/7.19.7 NSS/3.13.1.0 zlib/1.2.3 libidn/1.18 libssh2/1.2.2")
	request.Header.Add("Accept", "*/*")

	response, _ := client.Do(request)
	if response != nil {
		//fmt.Println(response.StatusCode, err)
	} else {
		return
	}
	defer response.Body.Close()

	flvHeader := make([]byte, HEADER_LEN)
	if _, err := io.ReadFull(response.Body, flvHeader); err != nil {
		return
	}
	if flvHeader[0] != 'F' || flvHeader[1] != 'L' || flvHeader[2] != 'V' {
		return
	}
	fmt.Println(string(flvHeader))

	for i := 0; ; i++ {
		header, data, err := ReadTag(response.Body)
		if err != nil {
			fmt.Println("ERRROR TAG\n")
			return
		}

		delay := getDelay(header, startTime)
		TotalBytes += len(data) / 1024
		if i%interval == 0 {
			fmt.Printf("[thread %v]Total TotalRecveived:%d KB, current Delay:%d ms\n", id, TotalBytes, delay)
		}

		//fmt.Printf("[thread %v]TagType:%v DataSize:%d Bytes\n", id, header.TagType, len(data))

		//if header.TagType == VIDEO_TAG && data[0] == KEY_FRAME {
		//	fmt.Println("FOUND KEY FRAME\n")
		//}
	}

}

func main() {
	var num int
	var reqUrl string

	flag.IntVar(&num, "n", 500, "thread num")
	flag.StringVar(&reqUrl, "url", "https://127.0.0.1:7001/live/movie.flv", "stream url")
	flag.Parse()

	for id := 0; id < num; id++ {
		//time.Sleep(time.Millisecond * 100)
		go httpFlv(reqUrl, id, 100)
	}
	time.Sleep(time.Hour)

}

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

	res := Get("http://127.0.0.1:8090/stat/livestat")
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

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}
