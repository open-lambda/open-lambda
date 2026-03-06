package event

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestMain(m *testing.M) {
	// Initialize common.Conf so that common.T0/T1 (latency tracking) doesn't panic
	common.Conf = &common.Config{}
	os.Exit(m.Run())
}

// --- Mocks ---

// MockKafkaClient implements KafkaClient for testing
type MockKafkaClient struct {
	pollFetchesFunc func(context.Context) kgo.Fetches
	closeCalled     bool
}

func (m *MockKafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	if m.pollFetchesFunc != nil {
		return m.pollFetchesFunc(ctx)
	}
	return kgo.Fetches{}
}

func (m *MockKafkaClient) Close() {
	m.closeCalled = true
}

// MockLambdaInvoker implements LambdaInvoker for testing
type MockLambdaInvoker struct {
	mu          sync.Mutex
	invocations []invokeRecord
}

type invokeRecord struct {
	LambdaName string
	Request    *http.Request
	Body       []byte
}

func (m *MockLambdaInvoker) Invoke(lambdaName string, w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invocations = append(m.invocations, invokeRecord{
		LambdaName: lambdaName,
		Request:    r,
		Body:       body,
	})
	w.WriteHeader(http.StatusOK)
}

func (m *MockLambdaInvoker) getInvocations() []invokeRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]invokeRecord, len(m.invocations))
	copy(cp, m.invocations)
	return cp
}

// --- Helper to build kgo.Fetches with records ---

func makeFetches(records ...*kgo.Record) kgo.Fetches {
	if len(records) == 0 {
		return kgo.Fetches{}
	}
	// Group records by topic+partition
	type key struct {
		topic     string
		partition int32
	}
	groups := map[key][]*kgo.Record{}
	for _, r := range records {
		k := key{r.Topic, r.Partition}
		groups[k] = append(groups[k], r)
	}

	topicMap := map[string][]kgo.FetchPartition{}
	for k, recs := range groups {
		topicMap[k.topic] = append(topicMap[k.topic], kgo.FetchPartition{
			Partition: k.partition,
			Records:   recs,
		})
	}

	var topics []kgo.FetchTopic
	for topic, partitions := range topicMap {
		topics = append(topics, kgo.FetchTopic{
			Topic:      topic,
			Partitions: partitions,
		})
	}
	return kgo.Fetches{{Topics: topics}}
}

func makeErrorFetches(topic string, partition int32, err error) kgo.Fetches {
	return kgo.Fetches{{
		Topics: []kgo.FetchTopic{{
			Topic: topic,
			Partitions: []kgo.FetchPartition{{
				Partition: partition,
				Err:       err,
			}},
		}},
	}}
}

// --- Tests ---

func TestConsumeLoop_ProcessesRecords(t *testing.T) {
	invoker := &MockLambdaInvoker{}

	record := &kgo.Record{
		Topic:     "orders",
		Partition: 3,
		Offset:    99,
		Value:     []byte(`{"orderId": 42}`),
	}

	var callCount atomic.Int32
	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			if callCount.Add(1) == 1 {
				return makeFetches(record)
			}
			return kgo.Fetches{}
		},
	}

	consumer := &LambdaKafkaConsumer{
		consumerName: "my-lambda-0",
		lambdaName:   "my-lambda",
		kafkaTrigger: &common.KafkaTrigger{GroupId: "lambda-my-lambda"},
		client:       mockClient,
		invoker:      invoker,
		stopChan:     make(chan struct{}),
	}

	go consumer.consumeLoop()
	time.Sleep(100 * time.Millisecond)
	close(consumer.stopChan)
	time.Sleep(50 * time.Millisecond)

	invocations := invoker.getInvocations()
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation, got %d", len(invocations))
	}

	inv := invocations[0]
	if inv.LambdaName != "my-lambda" {
		t.Errorf("Expected lambda name 'my-lambda', got %q", inv.LambdaName)
	}
	if string(inv.Body) != `{"orderId": 42}` {
		t.Errorf("Expected body '{\"orderId\": 42}', got %q", string(inv.Body))
	}
	if inv.Request.Method != "POST" {
		t.Errorf("Expected POST method, got %s", inv.Request.Method)
	}
	if inv.Request.URL.Path != "/run/my-lambda/" {
		t.Errorf("Expected path '/run/my-lambda/', got %q", inv.Request.URL.Path)
	}
	if inv.Request.RequestURI != "/run/my-lambda/" {
		t.Errorf("Expected RequestURI '/run/my-lambda/', got %q", inv.Request.RequestURI)
	}
	headers := map[string]string{
		"Content-Type":      "application/json",
		"X-Kafka-Topic":     "orders",
		"X-Kafka-Partition": "3",
		"X-Kafka-Offset":    "99",
		"X-Kafka-Group-Id":  "lambda-my-lambda",
	}
	for header, expected := range headers {
		if got := inv.Request.Header.Get(header); got != expected {
			t.Errorf("Header %s: expected %q, got %q", header, expected, got)
		}
	}
}

func TestConsumeLoop_ContinuesThroughErrors(t *testing.T) {
	// Capture logs so we can verify what gets logged
	var logBuf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	defer slog.SetDefault(old)

	invoker := &MockLambdaInvoker{}

	// Poll sequence: deadline-exceeded errors, then a real error, then a valid record.
	// The loop should survive both error types and still process the record.
	var callCount atomic.Int32
	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			n := callCount.Add(1)
			switch {
			case n <= 2:
				return makeErrorFetches("topic", 0, context.DeadlineExceeded)
			case n == 3:
				return makeErrorFetches("topic", 0, fmt.Errorf("broker unreachable"))
			case n == 4:
				return makeFetches(&kgo.Record{
					Topic: "topic", Partition: 0, Offset: 1, Value: []byte("survived"),
				})
			default:
				return kgo.Fetches{}
			}
		},
	}

	consumer := &LambdaKafkaConsumer{
		consumerName: "test-consumer",
		lambdaName:   "test-lambda",
		kafkaTrigger: &common.KafkaTrigger{GroupId: "test-group"},
		client:       mockClient,
		invoker:      invoker,
		stopChan:     make(chan struct{}),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.consumeLoop()
	}()
	time.Sleep(100 * time.Millisecond)
	close(consumer.stopChan)
	wg.Wait()

	invocations := invoker.getInvocations()
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation after errors, got %d", len(invocations))
	}
	if string(invocations[0].Body) != "survived" {
		t.Errorf("Expected body 'survived', got %q", string(invocations[0].Body))
	}

	// Real errors should be logged, but deadline exceeded should be silently skipped
	logs := logBuf.String()
	if !strings.Contains(logs, "broker unreachable") {
		t.Error("Expected 'broker unreachable' to be logged")
	}
	if strings.Contains(logs, "DeadlineExceeded") {
		t.Error("DeadlineExceeded should not appear in logs")
	}
}

func TestUnregister(t *testing.T) {
	manager := &KafkaManager{
		lambdaConsumers: make(map[string]*LambdaKafkaConsumer),
	}

	mockClient := &MockKafkaClient{}
	manager.lambdaConsumers["test-lambda-0"] = &LambdaKafkaConsumer{
		consumerName: "test-lambda-0",
		lambdaName:   "test-lambda",
		client:       mockClient,
		stopChan:     make(chan struct{}),
	}

	manager.UnregisterLambdaKafkaTriggers("test-lambda")

	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(manager.lambdaConsumers))
	}
	if !mockClient.closeCalled {
		t.Error("Expected Close to be called on client")
	}
}
