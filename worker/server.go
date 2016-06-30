package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/open-lambda/open-lambda/worker/handler"
	"github.com/open-lambda/open-lambda/worker/sandbox"
	"github.com/phonyphonecall/turnip"
)

type Config struct {
	RegistryHost string `json:"registry_host"`
	RegistryPort string `json:"registry_port"`
}

type Server struct {
	manager  sandbox.SandboxManager
	handlers *handler.HandlerSet

	// config options
	registry_host string
	registry_port string
	docker_host   string

	lambdaTimer *turnip.Turnip
}

type httpErr struct {
	msg  string
	code int
}

func newHttpErr(msg string, code int) *httpErr {
	return &httpErr{msg: msg, code: code}
}

func NewServer(
	registry_host string,
	registry_port string,
	docker_host string) (*Server, error) {

	// registry
	if registry_host == "" {
		registry_host = "localhost"
		log.Printf("Using '%v' for registry_host", registry_host)
	}

	if registry_port == "" {
		registry_port = "5000"
		log.Printf("Using '%v' for registry_port", registry_port)
	}

	// daemon
	cm := sandbox.NewDockerManager(registry_host, registry_port)
	if docker_host == "" {
		endpoint := cm.Client().Endpoint()
		local := "unix://"
		nonLocal := "https://"
		if strings.HasPrefix(endpoint, local) {
			docker_host = "localhost"
		} else if strings.HasPrefix(endpoint, nonLocal) {
			start := strings.Index(endpoint, nonLocal) + len([]rune(nonLocal))
			end := strings.LastIndex(endpoint, ":")
			docker_host = endpoint[start:end]
		} else {
			return nil, fmt.Errorf("please specify a docker host!")
		}
		log.Printf("Using '%v' for docker_host", docker_host)
	}

	// create server
	opts := handler.HandlerSetOpts{
		Cm:  cm,
		Lru: handler.NewHandlerLRU(100), // TODO(tyler)
	}
	server := &Server{
		registry_host: registry_host,
		registry_port: registry_port,
		docker_host:   docker_host,
		manager:       cm,
		handlers:      handler.NewHandlerSet(opts),
		lambdaTimer:   turnip.NewTurnip(),
	}

	return server, nil
}

func (s *Server) Manager() sandbox.SandboxManager {
	return s.manager
}

func (s *Server) ForwardToSandbox(handler *handler.Handler, r *http.Request, input []byte) ([]byte, *http.Response, *httpErr) {
	port, err := handler.RunStart()
	if err != nil {
		return nil, nil, newHttpErr(
			err.Error(),
			http.StatusInternalServerError)
	}

	defer handler.RunFinish()

	// forward request to sandbox.  r and w are the server
	// request and response respectively.  r2 and w2 are the
	// sandbox request and response respectively.
	host := fmt.Sprintf("%s:%s", s.docker_host, port)
	url := fmt.Sprintf("http://%s%s", host, r.URL.Path)
	// log.Printf("proxying request to %s\n", url)

	// TODO(tyler): some sort of smarter backoff.  Or, a better
	// way to detect a started sandbox.
	max_tries := 10
	for tries := 1; ; tries++ {
		r2, err := http.NewRequest(r.Method, url, bytes.NewReader(input))
		if err != nil {
			return nil, nil, newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}

		r2.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		client := &http.Client{}
		w2, err := client.Do(r2)
		if err != nil {
			log.Printf("request to sandbox failed with %v\n", err)
			if tries == max_tries {
				return nil, nil, newHttpErr(
					err.Error(),
					http.StatusInternalServerError)
			}
			log.Printf("retry request\n")
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
	handler := s.handlers.Get(img)
	wbody, w2, err := s.ForwardToSandbox(handler, r, rbody)
	if err != nil {
		return err
	}

	w.WriteHeader(w2.StatusCode)

	if _, err := w.Write(wbody); err != nil {
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
	// write response headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods",
		"GET, PUT, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers",
		"Content-Type, Content-Range, Content-Disposition, Content-Description, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
	} else {
		s.lambdaTimer.Start()
		if err := s.RunLambdaErr(w, r); err != nil {
			log.Printf("could not handle request: %s\n", err.msg)
			http.Error(w, err.msg, err.code)
		}
		s.lambdaTimer.Stop()

	}

}

func (s *Server) Dump() {
	log.Printf("============ Server Stats ===========\n")
	log.Printf("\tlambda: \t%fms\n", s.lambdaTimer.AverageMs())
	log.Printf("=====================================\n")
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

func main() {
	// parse config file
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <json-config>\n", os.Args[0])
	}
	config_raw, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("could not open config: %v\n", err.Error())
	}
	var config Config
	if err := json.Unmarshal(config_raw, &config); err != nil {
		log.Fatalf("could not parse config: %v\n", err.Error())
	}

	// start serving
	docker_host, ok := os.LookupEnv("OL_DOCKER_HOST")
	if !ok {
		docker_host = ""
	}
	server, err := NewServer(config.RegistryHost, config.RegistryPort, docker_host)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/runLambda/", server.RunLambda)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
