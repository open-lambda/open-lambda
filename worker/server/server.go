package server

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler"
	sbmanager "github.com/open-lambda/open-lambda/worker/sandbox-manager"
)

type Server struct {
	sbmanager sbmanager.SandboxManager
	config    *config.Config
	handlers  *handler.HandlerSet
}

type httpErr struct {
	msg  string
	code int
}

func newHttpErr(msg string, code int) *httpErr {
	return &httpErr{msg: msg, code: code}
}

func NewServer(config *config.Config) (*Server, error) {
	var sm sbmanager.SandboxManager
	var err error

	// Create sbmanager according to config
	if config.Registry == "docker" {
		sm, err = sbmanager.NewDockerManager(config)
	} else if config.Registry == "olregistry" {
		sm, err = sbmanager.NewRegistryManager(config)
	} else if config.Registry == "local" {
		sm, err = sbmanager.NewLocalManager(config)
	} else {
		return nil, errors.New("invalid 'registry' field in config")
	}

	if err != nil {
		return nil, err
	}

	// Create directory for pipes communicating with lambdas
	if err := os.Mkdir(filepath.Join(config.Worker_dir, "pipes"), 0700); err != nil {
		return nil, err
	}

	// Create named pipe to communicate with pre-initialized Python interpreter
	pipePath := filepath.Join(config.Worker_dir, "parent.pipe")
	err = syscall.Mkfifo(pipePath, 0666)
	if err != nil {
		return nil, err
	}

	// Find the location of the server.py script (TODO: better way to do this?)
	root, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}
	fmt.Print(root)
	scriptPath := path.Join(filepath.Dir(root), "lambda", "server.py")
	args := []string{"/usr/bin/python", scriptPath, pipePath}
	// TODO: how can lambda server print to stdout after forked?
	attr := os.ProcAttr{
		Files: []*os.File{nil, os.Stdout, os.Stderr},
	}

	_, err = os.StartProcess(args[0], args, &attr)
	if err != nil {
		return nil, err
	}

	opts := handler.HandlerSetOpts{
		Sm:     sm,
		Config: config,
		Lru:    handler.NewHandlerLRU(100), // TODO(tyler)
	}
	server := &Server{
		sbmanager: sm,
		config:    config,
		handlers:  handler.NewHandlerSet(opts),
	}

	return server, nil
}

func (s *Server) Manager() sbmanager.SandboxManager {
	return s.sbmanager
}

func (s *Server) ForwardToSandbox(handler *handler.Handler, r *http.Request, input []byte) ([]byte, *http.Response, *httpErr) {
	channel, err := handler.RunStart()
	if err != nil {
		return nil, nil, newHttpErr(
			err.Error(),
			http.StatusInternalServerError)
	}

	defer handler.RunFinish()

	// forward request to sandbox.  r and w are the server
	// request and response respectively.  r2 and w2 are the
	// sandbox request and response respectively.
	url := fmt.Sprintf("%s%s", channel.Url, r.URL.Path)

	// TODO(tyler): some sort of smarter backoff.  Or, a better
	// way to detect a started sandbox.
	max_tries := 10
	errors := []error{}
	for tries := 1; ; tries++ {
		r2, err := http.NewRequest(r.Method, url, bytes.NewReader(input))
		if err != nil {
			return nil, nil, newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}

		r2.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		client := &http.Client{Transport: &channel.Transport}
		w2, err := client.Do(r2)
		if err != nil {
			errors = append(errors, err)
			if tries == max_tries {
				log.Printf("Forwarding request to container failed after %v tries\n", max_tries)
				for i, item := range errors {
					log.Printf("Attempt %v: %v\n", i, item.Error())
				}
				return nil, nil, newHttpErr(
					err.Error(),
					http.StatusInternalServerError)
			}
			time.Sleep(time.Duration(tries*100) * time.Millisecond)
			continue
		}

		defer w2.Body.Close()
		wbody, err := ioutil.ReadAll(w2.Body)
		if err != nil {
			return nil, nil, newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}
		return wbody, w2, nil
	}
}

func (s *Server) RunLambdaErr(w http.ResponseWriter, r *http.Request) *httpErr {
	// components represent runLambda[0]/<name_of_sandbox>[1]/<extra_things>...
	// ergo we want [1] for name of sandbox
	urlParts := getUrlComponents(r)
	if len(urlParts) < 2 {
		return newHttpErr(
			"Name of image to run required",
			http.StatusBadRequest)
	}
	img := urlParts[1]
	i := strings.Index(img, "?")
	if i >= 0 {
		img = img[:i-1]
	}

	// read request
	rbody := []byte{}
	if r.Body != nil {
		defer r.Body.Close()
		var err error
		rbody, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}
	}

	// forward to sandbox
	// handler := s.handlers.Get(img)
	// wbody, w2, err := s.ForwardToSandbox(handler, r, rbody)
	// if err != nil {
	// 	return err
	// }

	handler := s.handlers.Get(img)
	_, err := handler.RunStart()
	if err != nil {
		return newHttpErr(err.Error(), http.StatusInternalServerError)
	}
	defer handler.RunFinish()

	pipePath := path.Join(s.config.Worker_dir, "pipes", img+".pipe")
	lambdaPipe, err := os.OpenFile(pipePath, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return newHttpErr(err.Error(), http.StatusInternalServerError)
	}
	lambdaPipe.Write(rbody)

	output, err := ioutil.ReadAll(lambdaPipe)
	if err != nil {
		return newHttpErr(err.Error(), http.StatusInternalServerError)
	}

	if _, err := w.Write(output); err != nil {
		return newHttpErr(
			err.Error(),
			http.StatusInternalServerError)
	}

	return nil
}

// RunLambda expects POST requests like this:
//
// curl -X POST localhost:8080/runLambda/<lambda-name> -d '{}'
func (s *Server) RunLambda(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	// write response headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods",
		"GET, PUT, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers",
		"Content-Type, Content-Range, Content-Disposition, Content-Description, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
	} else {
		if err := s.RunLambdaErr(w, r); err != nil {
			log.Printf("could not handle request: %s\n", err.msg)
			http.Error(w, err.msg, err.code)
		}
	}

}

func (s *Server) Status(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	wbody := []byte("ready")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Parses request URL into its "/" delimated components
func getUrlComponents(r *http.Request) []string {
	path := r.URL.Path

	// trim prefix
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// trim trailing "/"
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	components := strings.Split(path, "/")
	return components
}

func Main(config_path string) {
	log.Printf("Parse config\n")
	conf, err := config.ParseConfig(config_path)
	if err != nil {
		log.Fatal(err)
	}

	// start serving
	log.Printf("Create server\n")
	server, err := NewServer(conf)
	if err != nil {
		log.Fatal(err)
	}

	port := fmt.Sprintf(":%s", conf.Worker_port)
	run_path := "/runLambda/"
	status_path := "/status"
	http.HandleFunc(run_path, server.RunLambda)
	http.HandleFunc(status_path, server.Status)
	log.Printf("Execute handler by POSTing to localhost%s%s%s\n", port, run_path, "<lambda>")
	log.Printf("Get status by sending request to localhost%s%s\n", port, status_path)
	log.Fatal(http.ListenAndServe(port, nil))
}
