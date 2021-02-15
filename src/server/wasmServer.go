package server

import (
	"log"
	"net/http"
	"strings"
	"fmt"
	"io/ioutil"

	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

type WasmHandler func(http.ResponseWriter, []string, []byte) error

type WasmServer struct {
	engine *wasmer.Engine
	store *wasmer.Store
}

func (server *WasmServer) RunLambda(w http.ResponseWriter, rsrc []string, args []byte) error {
	wasmBytes, err := ioutil.ReadFile("test-registry.wasm/pyodide.asm.wasm")

	if err != nil {
		log.Fatal(err)
	}

	// TODO cache the module
	module, err := wasmer.NewModule(server.store, wasmBytes)

	if err != nil {
		log.Fatal(err)
	}

	importObject := makeEmscriptenBindings(server.store)

	instance, err := wasmer.NewInstance(module, importObject)

	if err == nil {
		log.Printf("Loaded and compiled wasm code")
	} else {
		log.Fatal(err)
	}

	content, err := ioutil.ReadFile(fmt.Sprintf("test-registry.wasm/%s", rsrc))
	if err != nil {
		log.Fatal(err)
	}
	
	// Convert []byte to string and print to screen
	code := string(content)
	
	log.Printf("Running code %s", code)

	loadFunc, _ := instance.Exports.GetFunction("loadPackagesFromIports")
	runFunc, _ := instance.Exports.GetFunction("runPython")

	loadFunc(code)
	runFunc(code)

	return nil
}

func (server *WasmServer) HandleInternal(w http.ResponseWriter, r *http.Request) error {
	log.Printf("%s %s", r.Method, r.URL.Path)

	defer r.Body.Close()

	if r.Method != "POST" {
		return fmt.Errorf("Only POST allowed (found %s)", r.Method)
	}

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	log.Printf("Body %s", rbody)

	rsrc := strings.Split(r.URL.Path, "/")
	if len(rsrc) < 2 {
		return fmt.Errorf("no path arguments provided in URL")
	}

	routes := map[string] WasmHandler{
		"run": server.RunLambda,
	}

	if h, ok := routes[rsrc[1]]; ok {
		return h(w, rsrc[2:], rbody)
	} else {
		return fmt.Errorf("unknown op %s", rsrc[1])
	}
}

func (server *WasmServer) Handle(w http.ResponseWriter, r *http.Request) {
	if err := server.HandleInternal(w, r); err != nil {
		log.Printf("Request Handler Failed: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%v\n", err)))
	}
}

func (server *WasmServer) cleanup() {
}

func NewWasmServer() (*WasmServer, error) {
	log.Printf("Starting WASM Server")

	//wasmBytes, _ := ioutil.ReadFile("pyodide.asm.wasm")

	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

    log.Printf("Created WASM engine")

	server := &WasmServer{ engine, store }

	http.HandleFunc("/", server.Handle)

	return server, nil
}
