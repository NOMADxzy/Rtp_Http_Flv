package configure

import (
	"flag"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const (
	MAX_UDP_CACHE_SIZE = 1024 * 1024 * 50 //50MB
)

var Log *logrus.Logger

type Config struct {
	h                             bool    `yaml:"h"`
	UDP_SOCKET_ADDR               string  `yaml:"udp_addr"`
	API_ADDR                      string  `yaml:"api_addr"`
	QUIC_ADDR                     string  `yaml:"quic_addr"`
	RTP_QUEUE_PADDING_WINDOW_SIZE int     `yaml:"padding_size"`
	DISABLE_QUIC                  bool    `yaml:"disable_quic"`
	RTP_QUEUE_CHAN_SIZE           int     `yaml:"chan_size"`
	RECORD_DIR                    string  `yaml:"record_dir"`
	PACKET_LOSS_RATE              float64 `yaml:"pack_loss"`
	HTTP_FLV_ADDR                 string  `yaml:"flv_addr"`
	ENABLE_HLS                    bool    `yaml:"enable_hls"`
	HLS_ADDR                      string  `yaml:"hls_addr"`
	ENABLE_RECORD                 bool    `yaml:"enable_record"`
	CERT_FILE                     string  `yaml:"cert_file"`
	KEY_FILE                      string  `yaml:"key_file"`
	INIT                          bool    `yaml:"init"`
	LOG_LEVEL                     string  `yaml:"log_level"`
	ENABLE_LOG_FILE               bool    `yaml:"enable_log_file"`
	CLOUD_HOST                    string  `yaml:"cloud_host"`
	PROTECT                       bool    `yaml:"protect"`
}

var Conf = &Config{
	false,
	"239.0.0.1:5222",
	":8090",
	":4242",
	100,
	false,
	100,
	"./record",
	0.001,
	":7001",
	false,
	":7002",
	false,
	"certs/server.crt",
	"certs/server.key",
	false, //保留值，不得手动设置
	"",
	false,
	"",   //保留值
	true, //保护模式，出现大面积连续丢包时会放弃重传这些包，跳到下个有效包
}

func init() {
	//初始化命令行参数
	flag.BoolVar(&Conf.h, "h", Conf.h, "this help")
	flag.StringVar(&Conf.UDP_SOCKET_ADDR, "udp_addr", Conf.UDP_SOCKET_ADDR, "udp listen addr") // :5222表示单播 239.0.0.1:5222表示组播
	flag.StringVar(&Conf.API_ADDR, "api_addr", Conf.API_ADDR, "http api server addr")          // 主要通过云端发过来，不建议在此指定
	flag.StringVar(&Conf.QUIC_ADDR, "quic_addr", Conf.QUIC_ADDR, "quic server addr")           // 主要通过云端发过来，不建议在此指定
	flag.IntVar(&Conf.RTP_QUEUE_PADDING_WINDOW_SIZE, "padding_size", Conf.RTP_QUEUE_PADDING_WINDOW_SIZE, "rtp queue window")
	flag.BoolVar(&Conf.DISABLE_QUIC, "disable_quic", Conf.DISABLE_QUIC, "enable quic service")
	flag.IntVar(&Conf.RTP_QUEUE_CHAN_SIZE, "chan_size", Conf.RTP_QUEUE_CHAN_SIZE, "rtp queue chan size")
	flag.StringVar(&Conf.RECORD_DIR, "record_dir", Conf.RECORD_DIR, "stream record dir")
	flag.Float64Var(&Conf.PACKET_LOSS_RATE, "pack_loss", Conf.PACKET_LOSS_RATE, "the rate to loss some packets")
	flag.StringVar(&Conf.HTTP_FLV_ADDR, "flv_addr", Conf.HTTP_FLV_ADDR, "HTTP-FLV server listen address")
	flag.BoolVar(&Conf.ENABLE_HLS, "enable_hls", Conf.ENABLE_HLS, "enable hls service")
	flag.StringVar(&Conf.HLS_ADDR, "hls_addr", Conf.HLS_ADDR, "HLS server listen address")
	flag.BoolVar(&Conf.ENABLE_RECORD, "enable_record", Conf.ENABLE_RECORD, "enable stream record")
	flag.StringVar(&Conf.CERT_FILE, "cert_file", Conf.CERT_FILE, "https server cert")
	flag.StringVar(&Conf.KEY_FILE, "key_file", Conf.KEY_FILE, "https server key")
	flag.StringVar(&Conf.LOG_LEVEL, "log_level", Conf.LOG_LEVEL, "log level")
	flag.BoolVar(&Conf.ENABLE_LOG_FILE, "enable_log_file", Conf.ENABLE_LOG_FILE, "enable log to file")
	flag.BoolVar(&Conf.PROTECT, "protect", Conf.PROTECT, "enable protect mode")

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

func InitConfig() bool { // 优先级 flag命令行 > config.yaml配置文件 > default默认配置
	Conf.readFromXml("./Config.yaml")

	flag.Parse() //获取参数
	if Conf.h {  //打印帮助
		flag.Usage()
		return false
	}

	//设置log
	Log = logrus.New()
	Log.Formatter = new(logrus.TextFormatter) //初始化log
	switch Conf.LOG_LEVEL {
	case "debug":
		Log.Level = logrus.DebugLevel
		break
	case "info":
		Log.Level = logrus.InfoLevel
		break
	case "error":
		Log.Level = logrus.ErrorLevel
		break
	default:
		Log.Level = logrus.TraceLevel
	}
	Log.Out = os.Stdout
	if Conf.ENABLE_LOG_FILE {
		file, err := os.OpenFile("main.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			Log.Out = file
		} else {
			Log.Info("Failed to log to file, using default stderr")
		}
	}

	//创建录制文件夹
	_, err := os.Stat(Conf.RECORD_DIR)
	if os.IsNotExist(err) {
		err := os.Mkdir(Conf.RECORD_DIR, os.ModePerm)
		if err != nil {
			panic(err)
		}
		Log.Infof("Create record dir: %v\n ", Conf.RECORD_DIR)
	}

	return true
}

func (Conf *Config) readFromXml(src string) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		Conf.writeToXml(src)
		return
	}
	err = yaml.Unmarshal(content, Conf)
	checkError(err)
}
func (Conf *Config) writeToXml(src string) {
	data, err := yaml.Marshal(Conf)
	checkError(err)
	err = ioutil.WriteFile(src, data, 0777)
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
