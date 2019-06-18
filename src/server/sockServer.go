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
	"strconv"
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

func (s *SOCKServer) GetSandbox(id string) sandbox.Sandbox {
	val, ok := s.sandboxes.Load(id)
	if !ok {
		return nil
	}
	return val.(sandbox.Sandbox)
}

func (s *SOCKServer) Create(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	// leaves are only in handler pool
	var pool *sandbox.SOCKPool
	if leaf, ok := args["leaf"]; !ok || leaf.(bool) {
		pool = s.handlerPool
	} else {
		pool = s.cachePool
	}

	// create args
	codeDir := args["code"].(string)

	scratchPrefix := filepath.Join(config.Conf.Worker_dir, "scratch")

	imports := []string{}

	var parent sandbox.Sandbox = nil
	if p, ok := args["parent"]; ok {
		parent = s.GetSandbox(p.(string))
	}

	// spin it up
	c, err := pool.CreateFromParent(codeDir, scratchPrefix, imports, parent)
	if err != nil {
		return err
	}
	s.sandboxes.Store(c.ID(), c)
	log.Printf("Save ID '%s' to map\n", c.ID())

	w.Write([]byte(fmt.Sprintf("%v\n", c.ID())))
	return nil
}

func (s *SOCKServer) Destroy(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	c := s.GetSandbox(rsrc[0])
	if c == nil {
		return fmt.Errorf("no sandbox found with ID '%s'", rsrc[0])
	}

	c.Destroy()

	return nil
}

func (s *SOCKServer) Pause(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	c := s.GetSandbox(rsrc[0])
	if c == nil {
		return fmt.Errorf("no sandbox found with ID '%s'", rsrc[0])
	}

	return c.Pause()
}

func (s *SOCKServer) Unpause(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	c := s.GetSandbox(rsrc[0])
	if c == nil {
		return fmt.Errorf("no sandbox found with ID '%s'", rsrc[0])
	}

	return c.Unpause()
}

func (s *SOCKServer) Debug(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	str := fmt.Sprintf(
		"CACHE SANDBOXES:\n\n%sHANDLER SANDBOXES:\n\n%s",
		s.cachePool.DebugString(), s.handlerPool.DebugString())
	fmt.Printf("%s\n", str)
	w.Write([]byte(str))
	return nil
}

// GetPid returns process ID, useful for making sure we're talking to the expected server
func (s *SOCKServer) GetPid(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	wbody := []byte(strconv.Itoa(os.Getpid()) + "\n")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
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

	routes := map[string]Handler{
		"create":  s.Create,
		"destroy": s.Destroy,
		"pause":   s.Pause,
		"unpause": s.Unpause,
		"debug":   s.Debug,
	}

	if h, ok := routes[rsrc[1]]; ok {
		return h(w, rsrc[2:], args)
	} else {
		return fmt.Errorf("unknown op %s", rsrc[1])
	}
}

func (s *SOCKServer) Handle(w http.ResponseWriter, r *http.Request) {
	if err := s.HandleInternal(w, r); err != nil {
		log.Printf("Request Handler Failed: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%v\n", err)))
	}
}

func (s *SOCKServer) cleanup() {
	s.cachePool.Cleanup()
	s.handlerPool.Cleanup()
}

func SockMain() {
	// start with a fresh env
	if err := os.RemoveAll(config.Conf.Worker_dir); err != nil {
		panic(err)
	}

	log.Printf("Start SOCK Server")
	server, err := NewSOCKServer()
	if err != nil {
		log.Printf("Could not create server")
		log.Fatal(err)
	}

	port := fmt.Sprintf(":%s", config.Conf.Worker_port)
	http.HandleFunc(PID_PATH, server.GetPid)
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
