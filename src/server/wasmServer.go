package server

import (
	"log"
	"net/http"
	"strings"
	"fmt"
)

type WasmServer struct {
}

func (s *WasmServer) HandleInternal(w http.ResponseWriter, r *http.Request) error {
	log.Printf("%s %s", r.Method, r.URL.Path)

	rsrc := strings.Split(r.URL.Path, "/")
	if len(rsrc) < 2 {
		return fmt.Errorf("no path arguments provided in URL")
	}

	return fmt.Errorf("unknown op %s", rsrc[1])

}

func (s *WasmServer) Handle(w http.ResponseWriter, r *http.Request) {
	if err := s.HandleInternal(w, r); err != nil {
		log.Printf("Request Handler Failed: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%v\n", err)))
	}
}

func (s *WasmServer) cleanup() {
}

func NewWasmServer() (*WasmServer, error) {
	log.Printf("Starting WASM Server")

	server := &WasmServer{}

	http.HandleFunc("/", server.Handle)

	return server, nil
}
