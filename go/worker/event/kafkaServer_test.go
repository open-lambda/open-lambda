package event

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestMain(m *testing.M) {
	// Initialize common.Conf so that common.T0/T1 (latency tracking) doesn't panic
	common.Conf = &common.Config{}
	os.Exit(m.Run())
}

// --- Mocks ---

// MockKafkaClient implements KafkaClient for testing.
//
// Instead of exposing pollFetchesFunc directly, tests enqueue responses via
// Send and SendError. The mock serves them in FIFO order and returns empty
// fetches once the queue is drained. This keeps polling/sequencing logic out
// of individual tests.
//
// The Drained channel (when set) is closed the first time PollFetches is called
// after the queue is empty. Because the consume loop calls PollFetches only
// after finishing the previous iteration's processing, a receive on Drained
// guarantees all enqueued records have been fully processed.
type MockKafkaClient struct {
	mu              sync.Mutex
	queue           []kgo.Fetches
	callCount       int
	closeCalled     atomic.Bool
	Drained         chan struct{} // closed when all queued fetches have been consumed and processed
	drainedSignaled bool
}

// Send enqueues records that will be returned by the next PollFetches call.
func (m *MockKafkaClient) Send(records ...*kgo.Record) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queue = append(m.queue, makeFetches(records...))
}

// SendError enqueues a fetch error that will be returned by the next PollFetches call.
func (m *MockKafkaClient) SendError(topic string, partition int32, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queue = append(m.queue, makeErrorFetches(topic, partition, err))
}

// PollFetches returns the next queued fetch, or empty fetches if the queue is
// drained. When the queue is empty and Drained is set, it closes Drained to
// signal that all prior records have been processed.
func (m *MockKafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callCount < len(m.queue) {
		f := m.queue[m.callCount]
		m.callCount++
		return f
	}
	if !m.drainedSignaled && m.Drained != nil {
		close(m.Drained)
		m.drainedSignaled = true
	}
	return kgo.Fetches{}
}

func (m *MockKafkaClient) Close() {
	m.closeCalled.Store(true)
}

// MockLambdaInvoker implements LambdaInvoker for testing.
type MockLambdaInvoker struct {
	mu          sync.Mutex
	invocations []invokeRecord
}

// invokeRecord captures the relevant fields from a lambda invocation in simple,
// comparable types. Tests can build an expected invokeRecord and compare it
// directly with reflect.DeepEqual instead of asserting each field individually.
type invokeRecord struct {
	LambdaName string
	Method     string
	Path       string
	RequestURI string
	Body       string
	Headers    map[string]string
}

func (m *MockLambdaInvoker) Invoke(lambdaName string, w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	// Flatten headers into a simple map for easy comparison in assertions
	headers := map[string]string{}
	for key := range r.Header {
		headers[key] = r.Header.Get(key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.invocations = append(m.invocations, invokeRecord{
		LambdaName: lambdaName,
		Method:     r.Method,
		Path:       r.URL.Path,
		RequestURI: r.RequestURI,
		Body:       string(body),
		Headers:    headers,
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

// --- Helpers ---

// makeFetches converts flat kgo.Records into the nested kgo.Fetches structure.
//
// franz-go's PollFetches returns a deeply nested type that mirrors how Kafka
// brokers organize data:
//
//	Fetches -> []Fetch -> []FetchTopic -> []FetchPartition -> []*Record
//
// Records are grouped by topic and then by partition. This helper handles that
// grouping automatically so tests can think in terms of simple records rather
// than the broker-level wire format.
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

// setupConsumerHarness creates the full test harness for exercising the consumer's
// consumeLoop. It mocks both sides of the consumer:
//
//   - Above the consumer (Kafka broker layer): MockKafkaClient replaces the real
//     Kafka connection so tests can enqueue records and errors without a broker.
//   - Below the consumer (lambda invocation layer): MockLambdaInvoker replaces the
//     real lambda invocation path so tests can capture and assert on HTTP requests.
//
// The consumer itself is real — it runs the actual consumeLoop logic, so tests
// exercise the full record-processing and error-handling pipeline.
func setupConsumerHarness(lambdaName string) (*MockKafkaClient, *MockLambdaInvoker, *LambdaKafkaConsumer) {
	// Mock above: fake Kafka broker
	client := &MockKafkaClient{Drained: make(chan struct{})}
	// Mock below: fake lambda invocation
	invoker := &MockLambdaInvoker{}

	consumer := &LambdaKafkaConsumer{
		consumerName: lambdaName + "-0",
		lambdaName:   lambdaName,
		kafkaTrigger: &common.KafkaTrigger{GroupId: "lambda-" + lambdaName},
		client:       client,
		invoker:      invoker,
		stopChan:     make(chan struct{}),
	}
	return client, invoker, consumer
}

// runConsumeLoop starts consumeLoop in a goroutine and returns a stop function
// that signals shutdown and waits for the goroutine to exit. Callers should
// <-mockClient.Drained before stop() — Drained closes once all enqueued
// records have been fully processed, making it safe to assert on results.
func runConsumeLoop(consumer *LambdaKafkaConsumer) (stop func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.consumeLoop()
	}()
	return func() {
		close(consumer.stopChan)
		wg.Wait()
	}
}

// --- Tests ---

func TestConsumeLoop_ProcessesRecords(t *testing.T) {
	mockClient, invoker, consumer := setupConsumerHarness("my-lambda")
	mockClient.Send(&kgo.Record{
		Topic: "orders", Partition: 3, Offset: 99,
		Value: []byte(`{"orderId": 42}`),
	})
	stop := runConsumeLoop(consumer)
	<-mockClient.Drained
	stop()

	invocations := invoker.getInvocations()
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation, got %d", len(invocations))
	}

	expected := invokeRecord{
		LambdaName: "my-lambda",
		Method:     "POST",
		Path:       "/run/my-lambda/",
		RequestURI: "/run/my-lambda/",
		Body:       `{"orderId": 42}`,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Kafka-Topic":     "orders",
			"X-Kafka-Partition": "3",
			"X-Kafka-Offset":    "99",
			"X-Kafka-Group-Id":  "lambda-my-lambda",
		},
	}
	if !reflect.DeepEqual(invocations[0], expected) {
		t.Errorf("Invocation mismatch:\n  got:  %+v\n  want: %+v", invocations[0], expected)
	}
}

func TestConsumeLoop_ContinuesThroughErrors(t *testing.T) {
	mockClient, invoker, consumer := setupConsumerHarness("test-lambda")
	// Poll sequence: deadline-exceeded errors (silently skipped), then a real
	// error (counted), then a valid record. The loop should survive all of them.
	mockClient.SendError("topic", 0, context.DeadlineExceeded)
	mockClient.SendError("topic", 0, context.DeadlineExceeded)
	mockClient.SendError("topic", 0, fmt.Errorf("broker unreachable"))
	mockClient.Send(&kgo.Record{
		Topic: "topic", Partition: 0, Offset: 1, Value: []byte("survived"),
	})
	stop := runConsumeLoop(consumer)
	<-mockClient.Drained
	stop()

	invocations := invoker.getInvocations()
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation after errors, got %d", len(invocations))
	}

	expected := invokeRecord{
		LambdaName: "test-lambda",
		Method:     "POST",
		Path:       "/run/test-lambda/",
		RequestURI: "/run/test-lambda/",
		Body:       "survived",
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Kafka-Topic":     "topic",
			"X-Kafka-Partition": "0",
			"X-Kafka-Offset":    "1",
			"X-Kafka-Group-Id":  "lambda-test-lambda",
		},
	}
	if !reflect.DeepEqual(invocations[0], expected) {
		t.Errorf("Invocation mismatch:\n  got:  %+v\n  want: %+v", invocations[0], expected)
	}

	// Only real errors should be counted; DeadlineExceeded should be ignored
	if consumer.errorCount != 1 {
		t.Errorf("Expected 1 error counted, got %d", consumer.errorCount)
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
	if !mockClient.closeCalled.Load() {
		t.Error("Expected Close to be called on client")
	}
}
