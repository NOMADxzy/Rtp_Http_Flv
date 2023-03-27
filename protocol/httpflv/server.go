package httpflv

import (
	"Rtp_Http_Flv/configure"
	"Rtp_Http_Flv/utils"
	"log"
	"net"
	"net/http"
	"strings"
)

type HttpHandler interface {
	HandleNewFlvWriter(key string, writer *FLVWriter)
	HandleDelayRequest(key string) int64
}

type Server struct {
	httpHandler HttpHandler
	//FLVWriterMap map[string]*FLVWriter
}

func NewServer(handler HttpHandler) *Server {
	return &Server{httpHandler: handler}
}

func (server *Server) Serve(l net.Listener, certFile string, keyFile string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server.handleConn(w, r)
	})

	mux.HandleFunc("/time", func(w http.ResponseWriter, r *http.Request) {
		server.handleTime(w, r)
	})

	if certFile == "" || keyFile == "" {
		log.Fatal(http.Serve(l, mux))
	} else {
		log.Fatal(http.ServeTLS(l, mux, certFile, keyFile))
	}
	return nil
}

func (server *Server) handleTime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
	server.httpHandler.HandleNewFlvWriter(path, writer)
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
