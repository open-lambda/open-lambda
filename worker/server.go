package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

type Server struct {
	manager *ContainerManager

	// config options
	registry_host string
	registry_port string
	docker_host string
}

func NewServer(
	registry_host string,
	registry_port string,
	docker_host string) (*Server, error) {

	// registry
	if registry_host == "" {
		registry_host = "localhost"
		log.Printf("Using %v for registry_host", registry_host)
	}

	if registry_port == "" {
		registry_port = "5000"
		log.Printf("Using %v for registry_port", registry_port)
	}

	// daemon
	cm := NewContainerManager(registry_host, registry_port)
	if docker_host == "" {
		if strings.HasPrefix(cm.Client().Endpoint(), "unix://") {
			docker_host = "localhost"
			log.Printf("Using %v for docker_host", docker_host)
		} else {
			return nil, fmt.Errorf("please specify a docker host!")
		}
	}

	// create server
	server := &Server{
		registry_host: registry_host,
		registry_port: registry_port,
		docker_host: docker_host,
		manager: cm,
	}
	return server, nil
}

// RunLambda expects POST requests like this:
//
// curl -X POST localhost:8080/runLambda/<lambda-name> -d '{}'
func (s *Server)RunLambda(w http.ResponseWriter, r *http.Request) {
	urlParts := getUrlComponents(r)
	if len(urlParts) < 2 {
		http.Error(w, "Name of image to run required", http.StatusBadRequest)
		return
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
		http.Error(w, "Failed to startup desired lambda", http.StatusInternalServerError)
		return
	}

	host := fmt.Sprintf("%s:%s", s.docker_host, port)

	log.Printf("proxying request to http://%s\n", host)
	director := func(req *http.Request) {
		req = r
		req.URL.Scheme = "http"
		req.URL.Host = host
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
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
	server,err := NewServer(os.Args[1], os.Args[2], docker_host)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/runLambda/", server.RunLambda)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
