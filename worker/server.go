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
	registry_host string
	registry_port string
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
	log.Printf("running lambda img \"%s\"\n", img)

	cm := NewContainerManager(s.registry_host, s.registry_port)

	// we'll ask docker manager to ensure the img is ready to accept requests
	// This will either start the img, or unpause a started one
	port, err := cm.DockerMakeReady(img)
	if err != nil {
		http.Error(w, "Failed to startup desired lambda", http.StatusInternalServerError)
		return
	}

	hostAddress, ok := os.LookupEnv("OL_DOCKER_HOST")
	if !ok {
		http.Error(w, "failed to lookup docker host (Did you set OL_DOCKER_HOST?)\n", http.StatusInternalServerError)
		return
	}

	host := fmt.Sprintf("%s:%s", hostAddress, port)

	log.Printf("proxying request to http://%s\n", host)
	director := func(req *http.Request) {
		req = r
		req.URL.Scheme = "http"
		req.URL.Host = host
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <registry hostname> <registry port>\n", os.Args[0])
	}

	server := Server{
		registry_host: os.Args[1],
		registry_port: os.Args[2],
	}

	http.HandleFunc("/runLambda/", server.RunLambda)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
