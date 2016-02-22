package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

// Parses request URL into its "/" delimated components
// Inspired by http://learntogoogleit.com/post/56844473263/url-path-to-array-in-golang
func getUrlComponents(r *http.Request) []string {
	path := r.URL.Path

	// trim beginning
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

func RunLambda(w http.ResponseWriter, r *http.Request) {
	urlParts := getUrlComponents(r)
	if len(urlParts) < 2 {
		http.Error(w, "Name of container image to run required", http.StatusBadRequest)
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

	// we'll ask docker manager to ensure the img is ready to accept requests
	// This will either start the img, or unpause a started one
	// Once this returns, the container is running on "host" ready for connections
	host, err := DockerMakeReady(img)
	if err != nil {
		http.Error(w, "Failed to startup desired lambda", http.StatusInternalServerError)
		return
	}

	var urlBuff bytes.Buffer
	urlBuff.WriteString(host)
	for _, part := range urlParts {
		urlBuff.WriteString("/")
		urlBuff.WriteString(part)
	}
	lambdaUrlString := urlBuff.String()
	log.Printf("proxying request to %s\n", lambdaUrlString)
	if err != nil {
		http.Error(w, "failed to create new lambda URL\n", http.StatusInternalServerError)
		return
	}

	director := func(req *http.Request) {
		req = r
		req.URL.Scheme = "http"
		req.URL.Host = lambdaUrlString
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <registry hostname> <registry port>\n", os.Args[0])
	}

	SetRegistry(os.Args[1], os.Args[2])

	http.HandleFunc("/runLambda/", RunLambda)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
