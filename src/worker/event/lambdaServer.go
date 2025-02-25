package event

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda"
)

// LambdaServer is a worker server that listens to run lambda requests and forward
// these requests to its sandboxes.
type LambdaServer struct {
	lambdaMgr *lambda.LambdaMgr
}

// Converts a kafka message to a HTTP request, since the LambdaServer Invocation and everything else relies on
// HTTP messages
func ConvertKafkaMessageToHTTPRequest(message []byte) (*http.Request, error) {
	// Define the target URL and HTTP method

	url := filepath.Join("/run", common.KafkaConf.Function) // Modify as needed
	method := "POST"

	// Create a new HTTP request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(message))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", "Kafka-Consumer-Client")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(message)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // Assuming Kafka messages are JSON

	return req, nil
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

func (s *LambdaServer) LaunchKafkaConsumer() {
	// olPath := common.Conf.Worker_dir
	// kafkaConfigPath := common.Conf.KafkaConfigPath
	// err := common.LoadKafkaConf(filepath.Join(olPath, kafkaConfigPath))
	// TODO: Get the path to the ol dir better
	err := common.LoadKafkaConf("/home/vboxuser/Documents/openl/open-lambda/myworker_k5/kafkaConfig.json")
	if err != nil {
		log.Fatalf("Failed to Load Kafka config: %s", err)
	}
	// Kafka consumer configuration
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": common.KafkaConf.Bootstrap_server,
		"group.id":          common.KafkaConf.Group_id,
		"auto.offset.reset": common.KafkaConf.Offset,
	})

	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
	}
	defer consumer.Close()

	topic := common.KafkaConf.Topic // Change this to your Kafka topic
	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %s", err)
	}

	log.Printf("Kafka consumer started. Listening for messages...")

	for {
		msg, err := consumer.ReadMessage(-1)
		if err == nil {
			r, _ := ConvertKafkaMessageToHTTPRequest(msg.Value)
			urlParts := getURLComponents(r)
			// TODO: This is because invoke requires a responseWriter, which is an interface.
			// Need an alternative, ideally without changing the whole Invocation struct
			w := httptest.NewRecorder()
			if len(urlParts) == 2 {
				img := urlParts[1]
				s.lambdaMgr.Get(img).Invoke(w, r)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("expected invocation format: /run/<lambda-name>"))
			}
		} else {
			log.Printf("Error while consuming: %v (%v)\n", err, msg)
		}
	}
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
func NewLambdaServer() (*LambdaServer, error) {
	log.Printf("Starting new lambda server")

	lambdaMgr, err := lambda.NewLambdaMgr()
	if err != nil {
		return nil, err
	}

	server := &LambdaServer{
		lambdaMgr: lambdaMgr,
	}
	switch common.Conf.Trigger {
	case "http":
		log.Printf("Setups Handlers")
		port := fmt.Sprintf(":%s", common.Conf.Worker_port)
		http.HandleFunc(RUN_PATH, server.RunLambda)
		http.HandleFunc(DEBUG_PATH, server.Debug)

		log.Printf("Execute handler by POSTing to localhost%s%s%s\n", port, RUN_PATH, "<lambda>")
		log.Printf("Get status by sending request to localhost%s%s\n", port, STATUS_PATH)
	case "kafka":
		server.LaunchKafkaConsumer()
	}

	return server, nil
}
