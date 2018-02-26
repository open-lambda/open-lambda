package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/benchmarker"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler"
)

const (
	RUN_PATH    = "/runLambda/"
	STATUS_PATH = "/status"
)

// Server is a worker server that listens to run lambda requests and forward
// these requests to its sandboxes.
type Server struct {
	config   *config.Config
	handlers *handler.HandlerManagerSet
}

// httpErr is a wrapper for an http error and the return code of the request.
type httpErr struct {
	msg  string
	code int
}

// newHttpErr creates an httpErr.
func newHttpErr(msg string, code int) *httpErr {
	return &httpErr{msg: msg, code: code}
}

// NewServer creates a server based on the passed config."
func NewServer(config *config.Config) (*Server, error) {
	handlers, err := handler.NewHandlerManagerSet(config)
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:   config,
		handlers: handlers,
	}

	return server, nil
}

// ForwardToSandbox forwards a run lambda request to a sandbox.
func (s *Server) ForwardToSandbox(handler *handler.Handler, r *http.Request, input []byte) ([]byte, *http.Response, error) {
	channel, err := handler.RunStart()
	if err != nil {
		return nil, nil, err
	}

	defer handler.RunFinish()

	if config.Timing {
		defer func(start time.Time) {
			log.Printf("forward request took %v\n", time.Since(start))
		}(time.Now())
	}

	// forward request to sandbox.  r and w are the server
	// request and response respectively.  r2 and w2 are the
	// sandbox request and response respectively.
	url := fmt.Sprintf("%s%s", channel.Url, r.URL.Path)

	// TODO(tyler): some sort of smarter backoff.  Or, a better
	// way to detect a started sandbox.
	max_tries := 10
	errors := []error{}
	for tries := 1; ; tries++ {
		b := benchmarker.GetBenchmarker()
		var t *benchmarker.Timer
		if b != nil {
			t = b.CreateTimer("lambda request", "us")
		}

		r2, err := http.NewRequest(r.Method, url, bytes.NewReader(input))
		if err != nil {
			return nil, nil, err
		}

		r2.Close = true
		r2.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		client := &http.Client{Transport: &channel.Transport}
		if t != nil {
			t.Start()
		}
		w2, err := client.Do(r2)
		if err != nil {
			if t != nil {
				t.Error("Request Failed")
			}
			errors = append(errors, err)
			if tries == max_tries {
				log.Printf("Forwarding request to container failed after %v tries\n", max_tries)
				for i, item := range errors {
					log.Printf("Attempt %v: %v\n", i, item.Error())
				}
				return nil, nil, err
			}
			time.Sleep(time.Duration(tries*100) * time.Millisecond)
			continue
		} else {
			if t != nil {
				t.End()
			}
		}

		defer w2.Body.Close()
		wbody, err := ioutil.ReadAll(w2.Body)
		if err != nil {
			return nil, nil, err
		}
		return wbody, w2, nil
	}
}

// RunLambdaErr handles the run lambda request and return an http error if any.
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
	var handler *handler.Handler
	if h, err := s.handlers.Get(img); err != nil {
		return newHttpErr(err.Error(), http.StatusInternalServerError)
	} else {
		handler = h
	}

	wbody, w2, err := s.ForwardToSandbox(handler, r, rbody)
	if err != nil {
		return newHttpErr(err.Error(), http.StatusInternalServerError)
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

// Status writes "ready" to the response.
func (s *Server) Status(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	wbody := []byte("ready\n")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	s.handlers.Dump()
}

// getUrlComponents parses request URL into its "/" delimated components
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

func (s *Server) cleanup() {
	s.handlers.Cleanup()
}

func Main(config_path string) {
	conf, err := config.ParseConfig(config_path)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Config: %+v", conf)

	server, err := NewServer(conf)
	if err != nil {
		log.Fatal(err)
	}

	if conf.Benchmark_file != "" {
		benchmarker.CreateBenchmarkerSingleton(conf.Benchmark_file)
	}

	port := fmt.Sprintf(":%s", conf.Worker_port)
	http.HandleFunc(RUN_PATH, server.RunLambda)
	http.HandleFunc(STATUS_PATH, server.Status)

	log.Printf("Execute handler by POSTing to localhost%s%s%s\n", port, RUN_PATH, "<lambda>")
	log.Printf("Get status by sending request to localhost%s%s\n", port, STATUS_PATH)

	// clean up if signal hits us
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func(s *Server) {
		<-c
		s.cleanup()
		os.Exit(1)
	}(server)

	log.Fatal(http.ListenAndServe(port, nil))
}
