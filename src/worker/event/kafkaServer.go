package event

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/ol/worker/lambda"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaServer struct {
	lambdaMgr *lambda.LambdaMgr
}

var kafkaServer *KafkaServer
var kafkaMu sync.Mutex

func ConvertKafkaMessageToHTTPRequest(message []byte) (*http.Request, error) {
	// Define the target URL and HTTP method

	url := filepath.Join("/run") // Modify as needed
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

func KafkaInit(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var data map[string]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Received config"))
	kafkaMu.Lock()
	if kafkaServer == nil {
		lambdaMgr, err := GetLambdaManagerInstance()
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}

		kafkaServer = &KafkaServer{
			lambdaMgr: lambdaMgr,
		}
	}
	kafkaMu.Unlock()
	seeds := strings.Split(data["bootstrap_servers"], ",")
	topics := strings.Split(data["topics"], ",")
	groupID := data["group_id"]
	functionName := data["functionName"]
	// Log or use them
	log.Printf("Starting Kafka server: Bootstrap: %s, Topic: %s, Group: %s, Function: %s",
		data["bootstrap_servers"], data["topics"], groupID, functionName)

	// One client can both produce and consume!
	// Consuming can either be direct (no consumer group), or through a group. Below, we use a group.
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(seeds...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
	)

	if err != nil {
		log.Printf("%f", err.Error())
		panic(err)
	}

	ctx := context.Background()
	go func() {
		for {
			fetches := cl.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				// All errors are retried internally when fetching, but non-retriable errors are
				// returned from polls so that users can notice and take action.
				panic(fmt.Sprint(errs))
			}

			// We can iterate through a record iterator...
			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				r, err := ConvertKafkaMessageToHTTPRequest(record.Value)
				if err != nil {
					log.Printf("%f", err.Error())
					panic(err)
				}
				w := httptest.NewRecorder()
				kafkaServer.lambdaMgr.Get(functionName).Invoke(w, r)
			}
		}
	}()
}
