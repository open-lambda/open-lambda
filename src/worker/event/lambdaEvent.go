package event

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda"
)

// LambdaEvent is a worker event that listens to run lambda requests and forward
// these requests to its sandboxes.
type LambdaEvent struct {
	lambdaMgr *lambda.LambdaMgr
}

// getURLComponents parses request URL into its "/" delimated components
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
func (s *LambdaEvent) RunLambda(w http.ResponseWriter, r *http.Request) {
	t := common.T0("web-request")
	defer t.T1()

	// TODO re-enable logging once it is configurable
	// log.Printf("Received request to %s\n", r.URL.Path)

	// components represent run[0]/<name_of_sandbox>[1]/<extra_things>...
	// ergo we want [1] for name of sandbox
	urlParts := getURLComponents(r)
	if len(urlParts) < 2 {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("expected invocation format: /run/<lambda-name>"))
	} else {
		// components represent run[0]/<name_of_sandbox>[1]/<extra_things>...
		// ergo we want [1] for name of sandbox
		urlParts := getURLComponents(r)
		if len(urlParts) == 2 {
			img := urlParts[1]
			s.lambdaMgr.Get(img).Invoke(w, r)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("expected invocation format: /run/<lambda-name>"))
		}
	}
}

func (s *LambdaEvent) Debug(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte(s.lambdaMgr.Debug()))
}

func (s *LambdaEvent) cleanup() {
	s.lambdaMgr.Cleanup()
}

// NewLambdaEvent creates a event based on the passed config."
func NewLambdaEvent() (*LambdaEvent, error) {
	log.Printf("Starting new lambda server")

	lambdaMgr, err := lambda.NewLambdaMgr()
	if err != nil {
		return nil, err
	}

	event := &LambdaEvent{
		lambdaMgr: lambdaMgr,
	}

	log.Printf("Setups Handlers")
	port := fmt.Sprintf(":%s", common.Conf.Worker_port)
	http.HandleFunc(RUN_PATH, event.RunLambda)
	http.HandleFunc(DEBUG_PATH, event.Debug)

	log.Printf("Execute handler by POSTing to localhost%s%s%s\n", port, RUN_PATH, "<lambda>")
	log.Printf("Get status by sending request to localhost%s%s\n", port, STATUS_PATH)

	return event, nil
}
