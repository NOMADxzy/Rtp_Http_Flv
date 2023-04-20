package configure

import (
	"flag"
	"fmt"
	"os"
)

const (
	MAX_UDP_CACHE_SIZE = 1024 * 1024 * 50 //50MB
)

var (
	h                             bool
	CLOUD_HOST                    string
	API_ADDR                      string
	UDP_SOCKET_ADDR               string
	QUIC_ADDR                     string
	RECORD_DIR                    string
	RTP_QUEUE_PADDING_WINDOW_SIZE int
	DISABLE_QUIC                  bool
	RTP_QUEUE_CHAN_SIZE           int
	PACKET_LOSS_RATE              float64
	HTTP_FLV_ADDR                 string
	ENABLE_HLS                    bool
	HLS_ADDR                      string
	ENABLE_RECORD                 bool
	CERT_FILE                     string
	KEY_FILE                      string
)

func init() {
	flag.BoolVar(&h, "h", false, "this help")
	//flag.StringVar(&CLOUD_HOST, "cloud_host", "127.0.0.1", "host of cloud server")
	flag.StringVar(&API_ADDR, "api_addr", ":8090", "http api server addr")
	flag.StringVar(&UDP_SOCKET_ADDR, "udp_addr", ":5222", "udp listen addr")
	flag.StringVar(&QUIC_ADDR, "quic_addr", ":4242", "quic server addr")
	flag.IntVar(&RTP_QUEUE_PADDING_WINDOW_SIZE, "padding_size", 100, "rtp queue window")
	flag.BoolVar(&DISABLE_QUIC, "disable_quic", false, "enable quic service")
	flag.IntVar(&RTP_QUEUE_CHAN_SIZE, "queue_chan_size", 100, "rtp queue chan size")
	flag.StringVar(&RECORD_DIR, "record_dir", "./record", "stream record dir")
	flag.Float64Var(&PACKET_LOSS_RATE, "pack_loss", 0.001, "the rate to loss some packets")
	flag.StringVar(&HTTP_FLV_ADDR, "httpflv_addr", ":7001", "HTTP-FLV server listen address")
	flag.BoolVar(&ENABLE_HLS, "enable_hls", false, "enable hls service")
	flag.StringVar(&HLS_ADDR, "hls_addr", ":7002", "HLS server listen address")
	flag.BoolVar(&ENABLE_RECORD, "enable_record", false, "enable stream record")
	flag.StringVar(&CERT_FILE, "cert_file", "certs/server.crt", "https server cert")
	flag.StringVar(&KEY_FILE, "key_file", "certs/server.key", "https server key")

	flag.Usage = usage
}

func usage() {
	//	fmt.Fprintf(os.Stderr, `nginx version: nginx/1.10.0
	//Usage: ./main [-api_url http api url] [-udp_addr udp address] [-quic_addr quic address] [-padding_size padding window size] [-enable_quic quic enable]
	//[-queue_chan_size queue in out channel size]
	//Options:
	//`)
	flag.PrintDefaults()
}

func GetFlag() bool {
	flag.Parse() //获取参数
	if h {       //打印帮助
		flag.Usage()
		return false
	}
	//创建录制文件夹
	_, err := os.Stat(RECORD_DIR)
	if os.IsNotExist(err) {
		err := os.Mkdir(RECORD_DIR, os.ModePerm)
		if err != nil {
			panic(err)
		}
		fmt.Println("Create record dir - ", RECORD_DIR)
	}

	return true
}
