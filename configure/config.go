package configure

import (
	"flag"
	"fmt"
	"os"
)

var (
	h                             bool
	API_URL                       string
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
)

func init() {
	flag.BoolVar(&h, "h", false, "this help")
	flag.StringVar(&API_URL, "api_url", "http://127.0.0.1:8090", "http api server addr")
	flag.StringVar(&UDP_SOCKET_ADDR, "udp_addr", ":5222", "udp listen addr")
	flag.StringVar(&QUIC_ADDR, "quic_addr", "127.0.0.1:4242", "quic server addr")
	flag.IntVar(&RTP_QUEUE_PADDING_WINDOW_SIZE, "padding_size", 300, "rtp queue window")
	flag.BoolVar(&DISABLE_QUIC, "disable_quic", false, "enable quic service")
	flag.IntVar(&RTP_QUEUE_CHAN_SIZE, "queue_chan_size", 100, "rtp queue chan size")
	flag.StringVar(&RECORD_DIR, "record_dir", "./record", "stream record dir")
	flag.Float64Var(&PACKET_LOSS_RATE, "pack_loss", 0.002, "the rate to loss some packets")
	flag.StringVar(&HTTP_FLV_ADDR, "httpflv_addr", ":7001", "HTTP-FLV server listen address")
	flag.BoolVar(&ENABLE_HLS, "enable_hls", true, "enable hls service")
	flag.StringVar(&HLS_ADDR, "hls_addr", ":7002", "HLS server listen address")

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

//func main() {
//	flag.Parse()
//	if h {
//		flag.Usage()
//	}
//}
