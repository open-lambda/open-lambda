package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/lambda"
)

// LambdaServer is a worker server that listens to run lambda requests and forward
// these requests to its sandboxes.
type LambdaServer struct {
	lambda_mgr *lambda.LambdaMgr
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

// RunLambda expects POST requests like this:
//
// curl -X POST localhost:8080/run/<lambda-name> -d '{}'
func (s *LambdaServer) RunLambda(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	// write response headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods",
		"GET, PUT, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers",
		"Content-Type, Content-Range, Content-Disposition, Content-Description, X-Requested-With")

	if r.Method == "OPTIONS" {
		// TODO: why not let the lambda decide?
		w.WriteHeader(http.StatusOK)
	} else {
		// components represent run[0]/<name_of_sandbox>[1]/<extra_things>...
		// ergo we want [1] for name of sandbox
		urlParts := getUrlComponents(r)
		if len(urlParts) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("expected invocation format: /run/<lambda-name>"))
		} else {
			img := urlParts[1]
			s.lambda_mgr.Get(img).Invoke(w, r)
		}
	}
}

func (s *LambdaServer) cleanup() {
	s.lambda_mgr.Cleanup()
}

// NewLambdaServer creates a server based on the passed config."
func NewLambdaServer() (*LambdaServer, error) {
	log.Printf("Start Lambda Server")

	lambda_mgr, err := lambda.NewLambdaMgr()
	if err != nil {
		return nil, err
	}

	server := &LambdaServer{
		lambda_mgr: lambda_mgr,
	}

	log.Printf("Setups Handlers")
	port := fmt.Sprintf(":%s", common.Conf.Worker_port)
	http.HandleFunc(RUN_PATH, server.RunLambda)

	log.Printf("Execute handler by POSTing to localhost%s%s%s\n", port, RUN_PATH, "<lambda>")
	log.Printf("Get status by sending request to localhost%s%s\n", port, STATUS_PATH)

	return server, nil
}
