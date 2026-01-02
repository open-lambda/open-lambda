package event

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/lambda"
)

// LambdaServer is a worker server that listens to run lambda requests and forward
// these requests to its sandboxes.
type LambdaServer struct {
	lambdaMgr *lambda.LambdaMgr
}

// getURLComponents parses request URL into its "/" delimited components
func getURLComponents(r *http.Request) []string {
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
// curl localhost:8080/run/<lambda-name>
// curl -X POST localhost:8080/run/<lambda-name> -d '{}'
// ...
func (s *LambdaServer) RunLambda(w http.ResponseWriter, r *http.Request) {
	t := common.T0("web-request")
	defer t.T1()

	// TODO re-enable logging once it is configurable
	// log.Printf("Received request to %s\n", r.URL.Path)

	// components represent run[0]/<name_of_sandbox>[1]/<extra_things>...
	// ergo we want [1] for name of sandbox
	urlParts := getURLComponents(r)

	if len(urlParts) < 2 {
		// client error, so we return bad request status
		http.Error(w, "expected invocation format: /run/<lambda-name>", http.StatusBadRequest)
		return
	}
	// Send entire path to app
	lambdaName := urlParts[1]
	f, err := s.lambdaMgr.Get(lambdaName)
	if err != nil {
		http.Error(w, "lambda server error: " + err.Error(), http.StatusInternalServerError)
		return
	}

	f.Invoke(w, r)
}

// Debug returns the debug information of the lambda manager.
func (s *LambdaServer) Debug(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte(s.lambdaMgr.Debug()))
}

// cleanup cleans up the lambda manager.
func (s *LambdaServer) cleanup() {
	s.lambdaMgr.Cleanup()
}

// NewLambdaServer creates a server based on the passed config.
func NewLambdaServer(mux *http.ServeMux) (*LambdaServer, error) {
	slog.Info("Starting new lambda server")

	lambdaMgr, err := lambda.GetLambdaManagerInstance()
	if err != nil {
		return nil, err
	}

	server := &LambdaServer{
		lambdaMgr: lambdaMgr,
	}

	slog.Info("Setups Handlers")
	port := fmt.Sprintf(":%s", common.Conf.Worker_port)
	mux.HandleFunc(RUN_PATH, server.RunLambda)
	mux.HandleFunc(DEBUG_PATH, server.Debug)

	slog.Info(fmt.Sprintf("Execute handler by POSTing to localhost%s%s%s", port, RUN_PATH, "<lambda>"))
	slog.Info(fmt.Sprintf("Get status by sending request to localhost%s%s", port, STATUS_PATH))

	return server, nil
}
