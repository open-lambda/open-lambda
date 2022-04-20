package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

type Handler func(http.ResponseWriter, []string, map[string]interface{}) error

var nextScratchId int64 = 1000

// SOCKServer is a worker server that listens to run lambda requests and forward
// these requests to its sandboxes.
type SOCKServer struct {
	sbPool    *sandbox.SOCKPool
	sandboxes sync.Map
}

func (server *SOCKServer) GetSandbox(id string) sandbox.Sandbox {
	val, ok := server.sandboxes.Load(id)
	if !ok {
		return nil
	}
	return val.(sandbox.Sandbox)
}

func (server *SOCKServer) Create(w http.ResponseWriter, rsrc []string, args map[string]interface{}) error {
	var leaf bool
	if b, ok := args["leaf"]; !ok || b.(bool) {
		leaf = true
	} else {
		leaf = false
	}

	// create args
	codeDir := args["code"].(string)

	var parent sandbox.Sandbox = nil
	if p, ok := args["parent"]; ok && p != "" {
		parent = server.GetSandbox(p.(string))
		if parent == nil {
			return fmt.Errorf("no sandbox found with ID '%s'", p)
		}
	}

	packages := []string{}
	if pkgs, ok := args["pkgs"]; ok {
		for _, p := range pkgs.([]interface{}) {
			packages = append(packages, p.(string))
		}
	}

	// spin it up
	scratchId := fmt.Sprintf("dir-%d", atomic.AddInt64(&nextScratchId, 1))
	scratchDir := filepath.Join(common.Conf.Worker_dir, "scratch", scratchId)
	if err := os.MkdirAll(scratchDir, 0777); err != nil {
		panic(err)
	}

	rt_type := common.RT_PYTHON

	if rt_name, ok := args["runtime"]; ok {
		if rt_name == "python" {
			rt_type = common.RT_PYTHON
		} else if rt_name == "native" {
			rt_type = common.RT_NATIVE
		} else {
			return fmt.Errorf("No such runtime `%s`", rt_name)
		}
	}

	if parent != nil && parent.GetRuntimeType() != rt_type {
		return fmt.Errorf("Parent and child have different runtimes")
	}

	meta := &sandbox.SandboxMeta{
		Installs: packages,
	}

	c, err := server.sbPool.Create(parent, leaf, codeDir, scratchDir, meta, rt_type)
	if err != nil {
		return err
	}

	server.sandboxes.Store(c.ID(), c)
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
	str := s.sbPool.DebugString()
	fmt.Printf("%s\n", str)
	w.Write([]byte(str))
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

	routes := map[string]Handler{
		"create":  s.Create,
		"destroy": s.Destroy,
		"pause":   s.Pause,
		"unpause": s.Unpause,
		"debug":   s.Debug,
	}

	if h, ok := routes[rsrc[1]]; ok {
		log.Printf("Got %s", rsrc[1])
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
	s.sandboxes.Range(func(key, val interface{}) bool {
		val.(sandbox.Sandbox).Destroy()
		return true
	})
	s.sbPool.Cleanup()
}

// NewSOCKServer creates a server based on the passed config."
func NewSOCKServer() (*SOCKServer, error) {
	log.Printf("Start SOCK Server")

	mem := sandbox.NewMemPool("sandboxes", common.Conf.Mem_pool_mb)
	sbPool, err := sandbox.NewSOCKPool("sandboxes", mem)
	if err != nil {
		return nil, err
	}
	// some of the SOCK tests depend on there not being an evictor
	// sandbox.NewSOCKEvictor(sbPool)

	server := &SOCKServer{
		sbPool: sbPool,
	}

	http.HandleFunc("/", server.Handle)

	return server, nil
}
