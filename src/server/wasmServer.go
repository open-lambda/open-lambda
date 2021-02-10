package server

import (
	"log"
	"net/http"
	"strings"
	"fmt"
	"io/ioutil"
	"encoding/json"

	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

type WasmServer struct {
	engine *wasmer.Engine
	store *wasmer.Store
}

func (s *WasmServer) HandleInternal(w http.ResponseWriter, r *http.Request) error {
	log.Printf("%s %s", r.Method, r.URL.Path)

	defer r.Body.Close()

	if r.Method != "POST" {
		return fmt.Errorf("Only POST allowed (found %s)", r.Method)
	}

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	var args map[string]interface{}

	if len(rbody) > 0 {
		if err := json.Unmarshal(rbody, &args); err != nil {
			return err
		}
	}

	log.Printf("Parsed Args: %v", args)

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

	//wasmBytes, _ := ioutil.ReadFile("pyodide.asm.wasm")

	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	server := &WasmServer{ engine, store }

	log.Printf("Loaded and compiled wasm code")

	http.HandleFunc("/", server.Handle)

	return server, nil
}
