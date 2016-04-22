package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/phonyphonecall/turnip"
)

type Server struct {
	manager *ContainerManager

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
	cm := NewContainerManager(registry_host, registry_port)
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
	server := &Server{
		registry_host: registry_host,
		registry_port: registry_port,
		docker_host:   docker_host,
		manager:       cm,
		lambdaTimer:   turnip.NewTurnip(),
	}

	return server, nil
}

func (s *Server) Manager() *ContainerManager {
	return s.manager
}

func (s *Server) RunLambdaErr(w http.ResponseWriter, r *http.Request) *httpErr {
	urlParts := getUrlComponents(r)
	if len(urlParts) < 2 {
		return newHttpErr(
			"Name of image to run required",
			http.StatusBadRequest)
	}

	// components represent runLambda[0]/<name_of_container>[1]/<extra_things>...
	// ergo we want [1] for name of container
	img := urlParts[1]
	i := strings.Index(img, "?")
	if i >= 0 {
		img = img[:i-1]
	}

	// we'll ask docker manager to ensure the img is ready to accept requests
	// This will either start the img, or unpause a started one
	port, err := s.manager.DockerMakeReady(img)
	if err != nil {
		return newHttpErr(
			err.Error(),
			http.StatusInternalServerError)
	}

	// read incoming request
	rbody := []byte{}
	if r.Body != nil {
		defer r.Body.Close()
		rbody, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}
	}

	// forward request to container.  r and w are the server
	// request and response respectively.  r2 and w2 are the
	// container request and response respectively.
	host := fmt.Sprintf("%s:%s", s.docker_host, port)
	url := fmt.Sprintf("http://%s%s", host, r.URL.Path)
	log.Printf("proxying request to %s\n", url)

	// TODO(tyler): some sort of smarter backoff.  Or, a better
	// way to detect a started container.
	for i := 0; i < 10; i++ {
		if i > 0 {
			log.Printf("retry request\n")
			time.Sleep(time.Duration(i*10) * time.Millisecond)
		}

		r2, err := http.NewRequest("POST", url, bytes.NewReader(rbody))
		if err != nil {
			return newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}
		r2.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		client := &http.Client{}
		w2, err := client.Do(r2)
		if err != nil {
			log.Printf("request to container failed with %v\n", err)
			continue
		}
		defer w2.Body.Close()
		wbody, err := ioutil.ReadAll(w2.Body)
		if err != nil {
			return newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}

		// forward response
		// TODO(tyler): origins should be configurable
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods",
			"GET, PUT, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Content-Range, Content-Disposition, Content-Description")

		w.WriteHeader(w2.StatusCode)
		if _, err := w.Write(wbody); err != nil {
			return newHttpErr(
				err.Error(),
				http.StatusInternalServerError)
		}
		return nil
	}

	return nil
}

// RunLambda expects POST requests like this:
//
// curl -X POST localhost:8080/runLambda/<lambda-name> -d '{}'
func (s *Server) RunLambda(w http.ResponseWriter, r *http.Request) {
	log.Printf("Ready\n")

	s.lambdaTimer.Start()
	if err := s.RunLambdaErr(w, r); err != nil {
		log.Printf("could not handle request: %s\n", err.msg)
		http.Error(w, err.msg, err.code)
	}
	s.lambdaTimer.Stop()
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
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <registry hostname> <registry port>\n", os.Args[0])
	}

	docker_host, ok := os.LookupEnv("OL_DOCKER_HOST")
	if !ok {
		docker_host = ""
	}
	server, err := NewServer(os.Args[1], os.Args[2], docker_host)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/runLambda/", server.RunLambda)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
