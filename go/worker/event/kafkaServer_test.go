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

// syncBuffer is a thread-safe bytes.Buffer for capturing logs from goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLogs redirects slog output to a thread-safe buffer for the duration of a test.
// Returns a function that reads the captured log output.
func captureLogs(t *testing.T) func() string {
	t.Helper()
	buf := &syncBuffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	old := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(old) })
	return func() string { return buf.String() }
}

func TestMain(m *testing.M) {
	// Initialize common.Conf so that common.T0/T1 (latency tracking) doesn't panic
	common.Conf = &common.Config{}
	os.Exit(m.Run())
}

// --- Mocks ---

// MockKafkaClient implements KafkaClient for testing
type MockKafkaClient struct {
	pollFetchesFunc func(context.Context) kgo.Fetches
	closeFunc       func()
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
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

// MockLambdaInvoker implements LambdaInvoker for testing
type MockLambdaInvoker struct {
	mu          sync.Mutex
	invocations []invokeRecord
	statusCode  int // status code to write back (default 200)
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
	code := m.statusCode
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)
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

// Initialization and Validation tests

func TestNewKafkaManager(t *testing.T) {
	manager, err := NewKafkaManager(nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
	if manager.lambdaConsumers == nil {
		t.Error("Expected lambdaConsumers map to be initialized")
	}
}

func TestNewLambdaKafkaConsumer_Validation(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	tests := []struct {
		name        string
		trigger     *common.KafkaTrigger
		expectedErr string
	}{
		{
			name: "empty bootstrap servers",
			trigger: &common.KafkaTrigger{
				BootstrapServers: []string{},
				Topics:           []string{"test-topic"},
			},
			expectedErr: "no bootstrap servers configured for lambda test-lambda",
		},
		{
			name: "empty topics",
			trigger: &common.KafkaTrigger{
				BootstrapServers: []string{"localhost:9092"},
				Topics:           []string{},
			},
			expectedErr: "no topics configured for lambda test-lambda",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.newLambdaKafkaConsumer("test-consumer", "test-lambda", tt.trigger)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("Expected error %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// Cleanup Tests (Consumer and Manager)

func TestConsumerCleanup(t *testing.T) {
	mockClient := &MockKafkaClient{}
	consumer := &LambdaKafkaConsumer{
		consumerName: "test-consumer",
		lambdaName:   "test-lambda",
		client:       mockClient,
		stopChan:     make(chan struct{}),
	}

	consumer.cleanup()

	if !mockClient.closeCalled {
		t.Error("Expected Close to be called on client")
	}

	select {
	case <-consumer.stopChan:
		// good — channel is closed
	default:
		t.Error("Expected stopChan to be closed")
	}
}

func TestManagerCleanup(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	clients := []*MockKafkaClient{{}, {}}
	for i, client := range clients {
		name := fmt.Sprintf("lambda%d-0", i)
		manager.lambdaConsumers[name] = &LambdaKafkaConsumer{
			consumerName: name,
			lambdaName:   fmt.Sprintf("lambda%d", i),
			client:       client,
			stopChan:     make(chan struct{}),
		}
	}

	manager.cleanup()

	for i, client := range clients {
		if !client.closeCalled {
			t.Errorf("Expected Close to be called on client %d", i)
		}
	}
	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers after cleanup, got %d", len(manager.lambdaConsumers))
	}
}

// Consumer Register / Unregister Tests

func TestRegisterNoTriggers(t *testing.T) {
	manager, _ := NewKafkaManager(nil)
	err := manager.RegisterLambdaKafkaTriggers("test-lambda", []common.KafkaTrigger{})
	if err != nil {
		t.Errorf("Expected no error for empty triggers, got %v", err)
	}
}

func TestRegisterReplacesExisting(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	oldClient := &MockKafkaClient{}
	manager.lambdaConsumers["test-lambda-0"] = &LambdaKafkaConsumer{
		consumerName: "test-lambda-0",
		lambdaName:   "test-lambda",
		client:       oldClient,
		stopChan:     make(chan struct{}),
	}

	// Register new triggers — will fail to create real kgo client,
	// but should still clean up the old consumer first
	triggers := []common.KafkaTrigger{{
		BootstrapServers: []string{"localhost:9092"},
		Topics:           []string{"new-topic"},
	}}
	manager.RegisterLambdaKafkaTriggers("test-lambda", triggers)

	if !oldClient.closeCalled {
		t.Error("Expected old client to be closed during re-registration")
	}
}

func TestUnregister(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

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

func TestUnregisterPreservesOtherLambdas(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	client1 := &MockKafkaClient{}
	client2 := &MockKafkaClient{}

	manager.lambdaConsumers["lambda1-0"] = &LambdaKafkaConsumer{
		consumerName: "lambda1-0",
		lambdaName:   "lambda1",
		client:       client1,
		stopChan:     make(chan struct{}),
	}
	manager.lambdaConsumers["lambda2-0"] = &LambdaKafkaConsumer{
		consumerName: "lambda2-0",
		lambdaName:   "lambda2",
		client:       client2,
		stopChan:     make(chan struct{}),
	}

	manager.UnregisterLambdaKafkaTriggers("lambda1")

	if len(manager.lambdaConsumers) != 1 {
		t.Errorf("Expected 1 consumer remaining, got %d", len(manager.lambdaConsumers))
	}
	if !client1.closeCalled {
		t.Error("Expected Close on lambda1 client")
	}
	if client2.closeCalled {
		t.Error("Expected lambda2 client to be untouched")
	}
	if _, exists := manager.lambdaConsumers["lambda2-0"]; !exists {
		t.Error("Expected lambda2-0 consumer to still exist")
	}
}

func TestConcurrentUnregister(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("lambda%d-0", i)
		manager.lambdaConsumers[name] = &LambdaKafkaConsumer{
			consumerName: name,
			lambdaName:   fmt.Sprintf("lambda%d", i),
			client:       &MockKafkaClient{},
			stopChan:     make(chan struct{}),
		}
	}

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(index int) {
			manager.UnregisterLambdaKafkaTriggers(fmt.Sprintf("lambda%d", index))
			done <- true
		}(i)
	}
	for i := 0; i < 5; i++ {
		<-done
	}

	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(manager.lambdaConsumers))
	}
}

// consumeLoop Tests

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
	if inv.Request.Header.Get("X-Kafka-Topic") != "orders" {
		t.Errorf("Expected X-Kafka-Topic 'orders', got %q", inv.Request.Header.Get("X-Kafka-Topic"))
	}
	if inv.Request.Header.Get("X-Kafka-Partition") != "3" {
		t.Errorf("Expected X-Kafka-Partition '3', got %q", inv.Request.Header.Get("X-Kafka-Partition"))
	}
	if inv.Request.Header.Get("X-Kafka-Offset") != "99" {
		t.Errorf("Expected X-Kafka-Offset '99', got %q", inv.Request.Header.Get("X-Kafka-Offset"))
	}
	if inv.Request.Header.Get("X-Kafka-Group-Id") != "lambda-my-lambda" {
		t.Errorf("Expected X-Kafka-Group-Id 'lambda-my-lambda', got %q", inv.Request.Header.Get("X-Kafka-Group-Id"))
	}
}

func TestConsumeLoop_SkipsDeadlineExceeded(t *testing.T) {
	getLogs := captureLogs(t)
	invoker := &MockLambdaInvoker{}

	// Poll sequence: 3 deadline-exceeded errors, then a real record, then empty.
	// The loop should silently skip all deadline errors and still process the record.
	var callCount atomic.Int32
	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			n := callCount.Add(1)
			if n <= 3 {
				return makeErrorFetches("topic", 0, context.DeadlineExceeded)
			}
			if n == 4 {
				return makeFetches(&kgo.Record{
					Topic: "topic", Partition: 0, Offset: 1, Value: []byte("survived"),
				})
			}
			return kgo.Fetches{}
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

	go consumer.consumeLoop()
	time.Sleep(100 * time.Millisecond)
	close(consumer.stopChan)
	time.Sleep(50 * time.Millisecond)

	// The record after the deadline errors should have been processed
	invocations := invoker.getInvocations()
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation after deadline errors, got %d", len(invocations))
	}
	if string(invocations[0].Body) != "survived" {
		t.Errorf("Expected body 'survived', got %q", string(invocations[0].Body))
	}

	// Deadline exceeded errors should NOT produce any warning logs
	logs := getLogs()
	if strings.Contains(logs, "Kafka fetch error") {
		t.Error("Deadline exceeded errors should be silently skipped, but got 'Kafka fetch error' in logs")
	}
}

func TestConsumeLoop_ContinuesOnRealErrors(t *testing.T) {
	getLogs := captureLogs(t)
	invoker := &MockLambdaInvoker{}

	// Poll sequence: a real error, then a valid record, then empty.
	// The loop should log the error as a warning, skip it, and still process the record.
	var callCount atomic.Int32
	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			n := callCount.Add(1)
			if n == 1 {
				return makeErrorFetches("topic", 0, fmt.Errorf("broker unreachable"))
			}
			if n == 2 {
				return makeFetches(&kgo.Record{
					Topic: "topic", Partition: 0, Offset: 1, Value: []byte("recovered"),
				})
			}
			return kgo.Fetches{}
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

	go consumer.consumeLoop()
	time.Sleep(100 * time.Millisecond)
	close(consumer.stopChan)
	time.Sleep(50 * time.Millisecond)

	// The record should still have been processed after the error
	invocations := invoker.getInvocations()
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation after error recovery, got %d", len(invocations))
	}
	if string(invocations[0].Body) != "recovered" {
		t.Errorf("Expected body 'recovered', got %q", string(invocations[0].Body))
	}

	// Real errors SHOULD produce a warning log (unlike deadline exceeded)
	logs := getLogs()
	if !strings.Contains(logs, "Kafka fetch error") {
		t.Error("Expected 'Kafka fetch error' warning in logs for real errors")
	}
	if !strings.Contains(logs, "broker unreachable") {
		t.Error("Expected 'broker unreachable' in log output")
	}
}

func TestConsumeLoop_StopsOnSignal(t *testing.T) {
	var callCount atomic.Int32
	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			callCount.Add(1)
			return kgo.Fetches{}
		},
	}

	consumer := &LambdaKafkaConsumer{
		consumerName: "test-consumer",
		lambdaName:   "test-lambda",
		kafkaTrigger: &common.KafkaTrigger{GroupId: "test-group"},
		client:       mockClient,
		invoker:      &MockLambdaInvoker{},
		stopChan:     make(chan struct{}),
	}

	go consumer.consumeLoop()
	time.Sleep(50 * time.Millisecond)
	close(consumer.stopChan)
	time.Sleep(50 * time.Millisecond)

	if callCount.Load() == 0 {
		t.Error("Expected PollFetches to be called at least once")
	}
}

// processMessage Tests

func TestProcessMessage(t *testing.T) {
	invoker := &MockLambdaInvoker{}

	consumer := &LambdaKafkaConsumer{
		consumerName: "test-consumer",
		lambdaName:   "my-func",
		kafkaTrigger: &common.KafkaTrigger{GroupId: "my-group"},
		invoker:      invoker,
		stopChan:     make(chan struct{}),
	}

	record := &kgo.Record{
		Topic:     "events",
		Partition: 7,
		Offset:    12345,
		Value:     []byte(`{"event": "click"}`),
	}

	consumer.processMessage(record)

	invocations := invoker.getInvocations()
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation, got %d", len(invocations))
	}

	inv := invocations[0]

	// Verify lambda name
	if inv.LambdaName != "my-func" {
		t.Errorf("Expected lambda name 'my-func', got %q", inv.LambdaName)
	}

	// Verify request path
	if inv.Request.URL.Path != "/run/my-func/" {
		t.Errorf("Expected path '/run/my-func/', got %q", inv.Request.URL.Path)
	}

	// Verify request method
	if inv.Request.Method != "POST" {
		t.Errorf("Expected POST method, got %s", inv.Request.Method)
	}

	// Verify body
	if string(inv.Body) != `{"event": "click"}` {
		t.Errorf("Expected body '{\"event\": \"click\"}', got %q", string(inv.Body))
	}

	// Verify all Kafka headers
	headers := map[string]string{
		"Content-Type":      "application/json",
		"X-Kafka-Topic":     "events",
		"X-Kafka-Partition": "7",
		"X-Kafka-Offset":    "12345",
		"X-Kafka-Group-Id":  "my-group",
	}
	for header, expected := range headers {
		if got := inv.Request.Header.Get(header); got != expected {
			t.Errorf("Header %s: expected %q, got %q", header, expected, got)
		}
	}

	// Verify RequestURI is set (needed for synthetic requests)
	if inv.Request.RequestURI != "/run/my-func/" {
		t.Errorf("Expected RequestURI '/run/my-func/', got %q", inv.Request.RequestURI)
	}
}

func TestProcessMessage_MultipleRecords(t *testing.T) {
	invoker := &MockLambdaInvoker{}

	consumer := &LambdaKafkaConsumer{
		consumerName: "test-consumer",
		lambdaName:   "processor",
		kafkaTrigger: &common.KafkaTrigger{GroupId: "grp"},
		invoker:      invoker,
		stopChan:     make(chan struct{}),
	}

	for i := 0; i < 5; i++ {
		consumer.processMessage(&kgo.Record{
			Topic:     "topic",
			Partition: 0,
			Offset:    int64(i),
			Value:     []byte(fmt.Sprintf("msg-%d", i)),
		})
	}

	invocations := invoker.getInvocations()
	if len(invocations) != 5 {
		t.Fatalf("Expected 5 invocations, got %d", len(invocations))
	}

	for i, inv := range invocations {
		expected := fmt.Sprintf("msg-%d", i)
		if string(inv.Body) != expected {
			t.Errorf("Invocation %d: expected body %q, got %q", i, expected, string(inv.Body))
		}
		if inv.LambdaName != "processor" {
			t.Errorf("Invocation %d: expected lambda name 'processor', got %q", i, inv.LambdaName)
		}
	}
}
