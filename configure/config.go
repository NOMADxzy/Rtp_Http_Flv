package configure

import (
	"flag"
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
)

func init() {
	flag.BoolVar(&h, "h", false, "this help")
	flag.StringVar(&API_URL, "api_url", "http://127.0.0.1:8090", "http api server addr")
	flag.StringVar(&UDP_SOCKET_ADDR, "udp_addr", "127.0.0.1:5222", "udp listen addr")
	flag.StringVar(&QUIC_ADDR, "quic_addr", "localhost:4242", "quic server addr")
	flag.IntVar(&RTP_QUEUE_PADDING_WINDOW_SIZE, "padding_size", 1000, "rtp queue window")
	flag.BoolVar(&DISABLE_QUIC, "disable_quic", false, "enable quic service")
	flag.IntVar(&RTP_QUEUE_CHAN_SIZE, "queue_chan_size", 100, "rtp queue chan size")
	flag.StringVar(&RECORD_DIR, "record_dir", "./record", "stream record dir")

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
	flag.Parse()
	if h {
		flag.Usage()
		return false
	}
	return true
}

//func main() {
//	flag.Parse()
//	if h {
//		flag.Usage()
//	}
//}
