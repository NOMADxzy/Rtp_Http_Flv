package httpflv

import (
	"Rtp_Http_Flv/configure"
	"log"
	"net"
	"net/http"
	"strings"
)

type HttpHandler interface {
	HandleNewFlvWriter(key string, writer *FLVWriter)
}

type Server struct {
	httpHandler HttpHandler
	//FLVWriterMap map[string]*FLVWriter
}

type stream struct {
	Key string `json:"key"`
	Id  string `json:"id"`
}

type streams struct {
	Publishers []stream `json:"publishers"`
	Players    []stream `json:"players"`
}

func NewServer(handler HttpHandler) *Server {
	return &Server{httpHandler: handler}
}

func (server *Server) Serve(l net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server.handleConn(w, r)
	})
	http.Serve(l, mux)
	return nil
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
		hdlServer.Serve(flvListen)
	}()
	return hdlServer
}
