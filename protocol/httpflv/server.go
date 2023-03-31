package httpflv

import (
	"Rtp_Http_Flv/configure"
	"Rtp_Http_Flv/utils"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
)

type HttpHandler interface {
	HandleNewFlvWriterRequest(key string, writer *FLVWriter)
	HandleDelayRequest(key string) (int64, error)
}

type Server struct {
	httpHandler HttpHandler
}

type Response struct {
	w      http.ResponseWriter
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
}

func (r *Response) SendJson() {
	resp, _ := json.Marshal(r)
	r.w.Header().Set("Content-Type", "application/json")
	r.w.WriteHeader(r.Status)
	_, err := r.w.Write(resp)
	utils.CheckError(err)
}

type TimeResult struct {
	StreamUrl string `json:"streamUrl"`
	StartTime int64  `json:"startTime"`
}

func NewServer(handler HttpHandler) *Server {
	return &Server{httpHandler: handler}
}

func (server *Server) Serve(l net.Listener, certFile string, keyFile string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server.handleConn(w, r)
	})

	mux.HandleFunc("/stats/time", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		utils.CheckError(err)
		path := r.Form.Get("stream")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		server.handleTime(w, path)
	})

	if certFile == "" || keyFile == "" {
		log.Fatal(http.Serve(l, mux))
	} else {
		log.Fatal(http.ServeTLS(l, mux, certFile, keyFile))
	}
	return nil
}

func (server *Server) handleTime(w http.ResponseWriter, path string) {
	startTime, _ := server.httpHandler.HandleDelayRequest(path)
	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}
	defer res.SendJson()
	if startTime == 0 {
		res.Status = 404
	} else {
		res.Data = TimeResult{StreamUrl: path, StartTime: startTime}
	}

}

func (server *Server) handleConn(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("http flv handleConn panic: ", r)
		}
	}()

	url := r.URL.String()
	u := r.URL.Path
	if pos := strings.LastIndex(u, "."); pos < 0 || u[pos:] != ".flv" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	path := strings.TrimSuffix(strings.TrimLeft(u, "/"), ".flv")
	paths := strings.SplitN(path, "/", 2)
	log.Println("url:", u, "path:", path, "paths:", paths)

	if len(paths) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	writer := NewFLVWriter(paths[0], paths[1], url, w)

	//server.handler.HandleWriter(writer)
	server.httpHandler.HandleNewFlvWriterRequest(path, writer)
	writer.Wait()
}
func StartHTTPFlv(handler HttpHandler) *Server {
	flvListen, err := net.Listen("tcp", configure.HTTP_FLV_ADDR)
	if err != nil {
		log.Fatal(err)
	}

	hdlServer := NewServer(handler)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("HTTP-FLV server panic: ", r)
			}
		}()
		log.Println("HTTP-FLV listen On", configure.HTTP_FLV_ADDR)
		//判断文件存在
		if !utils.PathExists(configure.KEY_FILE) || !utils.PathExists(configure.CERT_FILE) {
			configure.KEY_FILE = ""
			configure.CERT_FILE = ""
		}

		err := hdlServer.Serve(flvListen, configure.CERT_FILE, configure.KEY_FILE)
		utils.CheckError(err)
	}()
	return hdlServer
}
