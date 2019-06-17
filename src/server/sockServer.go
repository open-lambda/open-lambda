package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

type Handler func(http.ResponseWriter, []string, map[string]interface{}) error

// SOCKServer is a worker server that listens to run lambda requests and forward
// these requests to its sandboxes.
type SOCKServer struct {
	cachePool   *sandbox.SOCKPool
	handlerPool *sandbox.SOCKPool
	sandboxes   sync.Map
}

// NewSOCKServer creates a server based on the passed config."
func NewSOCKServer() (*SOCKServer, error) {
	cache, err := sandbox.NewSOCKPool(filepath.Join(config.Conf.Worker_dir, "cache-alone"), true)
	if err != nil {
		return nil, err
	}

	handler, err := sandbox.NewSOCKPool(filepath.Join(config.Conf.Worker_dir, "handler-alone"), false)
	if err != nil {
		return nil, err
	}

	server := &SOCKServer{
		cachePool:   cache,
		handlerPool: handler,
	}

	return server, nil
}

func (s *SOCKServer) GetSandbox(id string) *sandbox.SOCKContainer {
	val, ok := s.sandboxes.Load(id)
	if !ok {
		return nil
	}

	return val.(*sandbox.SOCKContainer)
}

func (s *SOCKServer) Create(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	// create args
	codeDir := args["code"].(string)
	workingDir, err := ioutil.TempDir("", "ol")
	if err != nil {
		return err
	}
	imports := []string{}
	var parent *sandbox.SOCKContainer

	log.Printf("CreateFromParent(%v, %v, %v, %v)", codeDir, workingDir, imports, parent)
	c, err := s.handlerPool.CreateFromParent(codeDir, workingDir, imports, parent)
	s.sandboxes.Store(c.ID(), c)
	log.Printf("Container: %v", c)

	w.Write([]byte(fmt.Sprintf("%v\n", c.ID())))

	return nil
}

func (s *SOCKServer) Destroy(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	c := s.GetSandbox(rsrc[0])
	if c == nil {
		return fmt.Errorf("no sandbox found with ID %s", rsrc[0])
	}

	c.Destroy()

	return nil
}

func (s *SOCKServer) HandleInternal(w http.ResponseWriter, r *http.Request) error {
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

	var h Handler

	switch rsrc[1] {
	case "create":
		h = s.Create
	case "destroy":
		h = s.Destroy
	default:
		return fmt.Errorf("unknown op %s", rsrc[1])
	}

	return h(w, rsrc[2:], args)
}

func (s *SOCKServer) Handle(w http.ResponseWriter, r *http.Request) {
	if err := s.HandleInternal(w, r); err != nil {
		log.Printf("Create Error: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%v\n", err)))
	}
}

func (s *SOCKServer) cleanup() {
	s.cachePool.Cleanup()
	s.handlerPool.Cleanup()
}

func SockMain() {
	log.Printf("Start SOCK Server")
	server, err := NewSOCKServer()
	if err != nil {
		log.Printf("Could not create server")
		log.Fatal(err)
	}

	port := fmt.Sprintf(":%s", config.Conf.Worker_port)
	http.HandleFunc("/", server.Handle)

	// clean up if signal hits us
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func(s *SOCKServer) {
		<-c
		log.Printf("received kill signal, cleaning up")
		s.cleanup()
		log.Printf("exiting")
		os.Exit(1)
	}(server)

	log.Fatal(http.ListenAndServe(port, nil))
}
