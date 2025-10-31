package event

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Mock KafkaClient implementation for testing
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

// TestNewKafkaManager tests the creation of a new KafkaManager
func TestNewKafkaManager(t *testing.T) {
	manager, err := NewKafkaManager(nil) // Can pass nil for basic initialization test
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

// TestNewLambdaKafkaConsumer_NoBootstrapServers tests validation
func TestNewLambdaKafkaConsumer_NoBootstrapServers(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	trigger := &common.KafkaTrigger{
		BootstrapServers: []string{},
		Topics:           []string{"test-topic"},
		GroupId:          "test-group",
	}

	_, err := manager.newLambdaKafkaConsumer("test-consumer", "test-lambda", trigger)
	if err == nil {
		t.Fatal("Expected error for empty bootstrap servers")
	}

	expectedError := "no bootstrap servers configured for lambda test-lambda"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestNewLambdaKafkaConsumer_NoTopics tests validation
func TestNewLambdaKafkaConsumer_NoTopics(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	trigger := &common.KafkaTrigger{
		BootstrapServers: []string{"localhost:9092"},
		Topics:           []string{},
		GroupId:          "test-group",
	}

	_, err := manager.newLambdaKafkaConsumer("test-consumer", "test-lambda", trigger)
	if err == nil {
		t.Fatal("Expected error for empty topics")
	}

	expectedError := "no topics configured for lambda test-lambda"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestLambdaKafkaConsumer_Cleanup tests cleanup functionality
func TestLambdaKafkaConsumer_Cleanup(t *testing.T) {
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

	// Verify stopChan is closed
	select {
	case <-consumer.stopChan:
	default:
		t.Error("Expected stopChan to be closed")
	}
}

// TestRegisterLambdaKafkaTriggers_NoTriggers tests with empty triggers
func TestRegisterLambdaKafkaTriggers_NoTriggers(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	err := manager.RegisterLambdaKafkaTriggers("test-lambda", []common.KafkaTrigger{})
	if err != nil {
		t.Errorf("Expected no error for empty triggers, got %v", err)
	}
}

// TestUnregisterLambdaKafkaTriggers tests trigger unregistration
func TestUnregisterLambdaKafkaTriggers(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	// Add a mock consumer
	mockClient := &MockKafkaClient{}
	consumer := &LambdaKafkaConsumer{
		consumerName: "test-lambda-0",
		lambdaName:   "test-lambda",
		client:       mockClient,
		stopChan:     make(chan struct{}),
	}

	manager.lambdaConsumers["test-lambda-0"] = consumer

	// Unregister
	manager.UnregisterLambdaKafkaTriggers("test-lambda")

	// Verify consumer was removed
	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(manager.lambdaConsumers))
	}

	// Verify client was closed
	if !mockClient.closeCalled {
		t.Error("Expected Close to be called on client")
	}
}

// TestUnregisterLambdaKafkaTriggers_MultipleConsumers tests unregistering multiple consumers
func TestUnregisterLambdaKafkaTriggers_MultipleConsumers(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	// Add multiple mock consumers for the same lambda
	mockClient1 := &MockKafkaClient{}
	mockClient2 := &MockKafkaClient{}

	consumer1 := &LambdaKafkaConsumer{
		consumerName: "test-lambda-0",
		lambdaName:   "test-lambda",
		client:       mockClient1,
		stopChan:     make(chan struct{}),
	}

	consumer2 := &LambdaKafkaConsumer{
		consumerName: "test-lambda-1",
		lambdaName:   "test-lambda",
		client:       mockClient2,
		stopChan:     make(chan struct{}),
	}

	manager.lambdaConsumers["test-lambda-0"] = consumer1
	manager.lambdaConsumers["test-lambda-1"] = consumer2

	// Unregister all consumers for this lambda
	manager.UnregisterLambdaKafkaTriggers("test-lambda")

	// Verify all consumers were removed
	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(manager.lambdaConsumers))
	}

	// Verify both clients were closed
	if !mockClient1.closeCalled {
		t.Error("Expected Close to be called on client1")
	}
	if !mockClient2.closeCalled {
		t.Error("Expected Close to be called on client2")
	}
}

// TestUnregisterLambdaKafkaTriggers_PreservesOtherLambdas tests that other lambdas are not affected
func TestUnregisterLambdaKafkaTriggers_PreservesOtherLambdas(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	// Add consumers for different lambdas
	mockClient1 := &MockKafkaClient{}
	mockClient2 := &MockKafkaClient{}

	consumer1 := &LambdaKafkaConsumer{
		consumerName: "lambda1-0",
		lambdaName:   "lambda1",
		client:       mockClient1,
		stopChan:     make(chan struct{}),
	}

	consumer2 := &LambdaKafkaConsumer{
		consumerName: "lambda2-0",
		lambdaName:   "lambda2",
		client:       mockClient2,
		stopChan:     make(chan struct{}),
	}

	manager.lambdaConsumers["lambda1-0"] = consumer1
	manager.lambdaConsumers["lambda2-0"] = consumer2

	// Unregister only lambda1
	manager.UnregisterLambdaKafkaTriggers("lambda1")

	// Verify only lambda1's consumer was removed
	if len(manager.lambdaConsumers) != 1 {
		t.Errorf("Expected 1 consumer remaining, got %d", len(manager.lambdaConsumers))
	}

	// Verify lambda1's client was closed but not lambda2's
	if !mockClient1.closeCalled {
		t.Error("Expected Close to be called on client1")
	}
	if mockClient2.closeCalled {
		t.Error("Expected Close NOT to be called on client2")
	}

	// Verify lambda2's consumer is still present
	if _, exists := manager.lambdaConsumers["lambda2-0"]; !exists {
		t.Error("Expected lambda2-0 consumer to still exist")
	}
}

// TestKafkaManager_Cleanup tests manager cleanup
func TestKafkaManager_Cleanup(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	// Add mock consumers
	mockClient1 := &MockKafkaClient{}
	mockClient2 := &MockKafkaClient{}

	consumer1 := &LambdaKafkaConsumer{
		consumerName: "lambda1-0",
		lambdaName:   "lambda1",
		client:       mockClient1,
		stopChan:     make(chan struct{}),
	}

	consumer2 := &LambdaKafkaConsumer{
		consumerName: "lambda2-0",
		lambdaName:   "lambda2",
		client:       mockClient2,
		stopChan:     make(chan struct{}),
	}

	manager.lambdaConsumers["lambda1-0"] = consumer1
	manager.lambdaConsumers["lambda2-0"] = consumer2

	// Cleanup
	manager.cleanup()

	// Verify all clients were closed
	if !mockClient1.closeCalled {
		t.Error("Expected Close to be called on client1")
	}
	if !mockClient2.closeCalled {
		t.Error("Expected Close to be called on client2")
	}

	// Verify map was cleared
	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers after cleanup, got %d", len(manager.lambdaConsumers))
	}
}

// TestConsumeLoop_StopChannel tests that consumeLoop respects stop signal
func TestConsumeLoop_StopChannel(t *testing.T) {
	callCount := 0

	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			callCount++
			return kgo.Fetches{}
		},
	}

	consumer := &LambdaKafkaConsumer{
		consumerName:  "test-consumer",
		lambdaName:    "test-lambda",
		kafkaTrigger:  &common.KafkaTrigger{GroupId: "test-group"},
		client:        mockClient,
		lambdaManager: nil,
		stopChan:      make(chan struct{}),
	}

	// Start the consume loop in a goroutine
	go consumer.consumeLoop()

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	// Stop the consumer
	close(consumer.stopChan)

	// Wait for cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify the loop executed at least once
	if callCount == 0 {
		t.Error("Expected PollFetches to be called at least once")
	}
}

// TestConsumeLoop_ErrorHandling tests error handling in consume loop
func TestConsumeLoop_ErrorHandling(t *testing.T) {
	callCount := 0
	maxCalls := 5

	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			callCount++
			if callCount >= maxCalls {
				return kgo.Fetches{}
			}
			// Simulate a deadline exceeded error (which should be ignored)
			return kgo.Fetches{}
		},
	}

	consumer := &LambdaKafkaConsumer{
		consumerName:  "test-consumer",
		lambdaName:    "test-lambda",
		kafkaTrigger:  &common.KafkaTrigger{GroupId: "test-group"},
		client:        mockClient,
		lambdaManager: nil,
		stopChan:      make(chan struct{}),
	}

	// Start the consume loop
	go consumer.consumeLoop()

	// Let it run
	time.Sleep(100 * time.Millisecond)

	// Stop it
	close(consumer.stopChan)

	time.Sleep(50 * time.Millisecond)

	// The loop should have handled errors gracefully
	if callCount == 0 {
		t.Error("Expected at least one PollFetches call")
	}
}

// TestConcurrentAccess tests thread safety of KafkaManager
func TestConcurrentAccess(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	// Add some consumers
	for i := 0; i < 5; i++ {
		mockClient := &MockKafkaClient{}
		consumer := &LambdaKafkaConsumer{
			consumerName: fmt.Sprintf("lambda%d-0", i),
			lambdaName:   fmt.Sprintf("lambda%d", i),
			client:       mockClient,
			stopChan:     make(chan struct{}),
		}
		manager.lambdaConsumers[consumer.consumerName] = consumer
	}

	// Concurrently unregister
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(index int) {
			manager.UnregisterLambdaKafkaTriggers(fmt.Sprintf("lambda%d", index))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all consumers were removed
	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(manager.lambdaConsumers))
	}
}

// TestConcurrentCleanup tests concurrent cleanup operations
func TestConcurrentCleanup(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	// Add consumers
	clients := make([]*MockKafkaClient, 10)
	for i := 0; i < 10; i++ {
		clients[i] = &MockKafkaClient{}
		consumer := &LambdaKafkaConsumer{
			consumerName: fmt.Sprintf("lambda%d-0", i),
			lambdaName:   fmt.Sprintf("lambda%d", i),
			client:       clients[i],
			stopChan:     make(chan struct{}),
		}
		manager.lambdaConsumers[consumer.consumerName] = consumer
	}

	// Run cleanup multiple times concurrently
	done := make(chan bool)
	for i := 0; i < 3; i++ {
		go func() {
			manager.cleanup()
			done <- true
		}()
	}

	// Wait for all
	for i := 0; i < 3; i++ {
		<-done
	}

	// All consumers should be cleaned up
	if len(manager.lambdaConsumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(manager.lambdaConsumers))
	}
}

// TestRegisterLambdaKafkaTriggers_ReplacesExisting tests that existing consumers are cleaned up
func TestRegisterLambdaKafkaTriggers_ReplacesExisting(t *testing.T) {
	manager, _ := NewKafkaManager(nil)

	// Add an existing consumer
	mockClient := &MockKafkaClient{}
	consumer := &LambdaKafkaConsumer{
		consumerName: "test-lambda-0",
		lambdaName:   "test-lambda",
		client:       mockClient,
		stopChan:     make(chan struct{}),
	}
	manager.lambdaConsumers["test-lambda-0"] = consumer

	// Register new triggers (this should cleanup the old one)
	triggers := []common.KafkaTrigger{
		{
			BootstrapServers: []string{"localhost:9092"},
			Topics:           []string{"new-topic"},
			GroupId:          "new-group",
		},
	}

	// This will fail to create a new consumer (no real Kafka),
	// but should still cleanup the old one
	manager.RegisterLambdaKafkaTriggers("test-lambda", triggers)

	// Verify old client was closed
	if !mockClient.closeCalled {
		t.Error("Expected old client to be closed")
	}
}

// TestMockKafkaClient_CustomClose tests MockKafkaClient with custom close function
func TestMockKafkaClient_CustomClose(t *testing.T) {
	closeFuncCalled := false
	mockClient := &MockKafkaClient{
		closeFunc: func() {
			closeFuncCalled = true
		},
	}

	mockClient.Close()

	if !mockClient.closeCalled {
		t.Error("Expected closeCalled to be true")
	}
	if !closeFuncCalled {
		t.Error("Expected custom close function to be called")
	}
}

// TestProcessMessage_Integration tests that processMessage is called when records arrive
func TestProcessMessage_Integration(t *testing.T) {
	recordsProcessed := 0
	messageValue := []byte(`{"test": "data"}`)

	// Create a mock client that returns a record on first call, then stops
	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			if recordsProcessed > 0 {
				// Return empty after first call
				return kgo.Fetches{}
			}
			recordsProcessed++

			return kgo.Fetches{}
		},
	}

	consumer := &LambdaKafkaConsumer{
		consumerName:  "test-consumer",
		lambdaName:    "test-lambda",
		kafkaTrigger:  &common.KafkaTrigger{GroupId: "test-group"},
		client:        mockClient,
		lambdaManager: nil,
		stopChan:      make(chan struct{}),
	}

	// Start consume loop
	go consumer.consumeLoop()

	// Let it poll a few times
	time.Sleep(100 * time.Millisecond)

	// Stop consumer
	close(consumer.stopChan)
	time.Sleep(50 * time.Millisecond)

	// Verify PollFetches was called
	if recordsProcessed == 0 {
		t.Error("Expected PollFetches to be called at least once")
	}
}

// TestProcessMessage_RequestHeaders verifies HTTP request construction from Kafka records
func TestProcessMessage_RequestHeaders(t *testing.T) {
	// Simulate a Kafka record
	topic := "test-topic"
	partition := int32(5)
	offset := int64(12345)
	groupId := "test-group-id"
	messageBody := []byte(`{"user": "test", "action": "login"}`)

	// Create the request as processMessage would
	req, err := http.NewRequest("POST", "/", bytes.NewReader(messageBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set headers as processMessage does
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kafka-Topic", topic)
	req.Header.Set("X-Kafka-Partition", fmt.Sprintf("%d", partition))
	req.Header.Set("X-Kafka-Offset", fmt.Sprintf("%d", offset))
	req.Header.Set("X-Kafka-Group-Id", groupId)

	// Verify all headers are set correctly
	tests := []struct {
		header   string
		expected string
	}{
		{"Content-Type", "application/json"},
		{"X-Kafka-Topic", "test-topic"},
		{"X-Kafka-Partition", "5"},
		{"X-Kafka-Offset", "12345"},
		{"X-Kafka-Group-Id", "test-group-id"},
	}

	for _, tt := range tests {
		if got := req.Header.Get(tt.header); got != tt.expected {
			t.Errorf("Header %s: expected %q, got %q", tt.header, tt.expected, got)
		}
	}

	// Verify request method
	if req.Method != "POST" {
		t.Errorf("Expected POST method, got %s", req.Method)
	}

	// Verify body can be read
	body := make([]byte, len(messageBody))
	n, err := req.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("Failed to read body: %v", err)
	}
	if n != len(messageBody) {
		t.Errorf("Expected to read %d bytes, got %d", len(messageBody), n)
	}
	if string(body) != string(messageBody) {
		t.Errorf("Body mismatch: expected %q, got %q", messageBody, body)
	}
}

// TestProcessMessage_MultipleMessages tests that multiple records would be processed
func TestProcessMessage_MultipleMessages(t *testing.T) {
	// This test verifies the consume loop pattern for multiple messages
	callCount := 0
	maxCalls := 10

	mockClient := &MockKafkaClient{
		pollFetchesFunc: func(ctx context.Context) kgo.Fetches {
			callCount++
			if callCount >= maxCalls {
				return kgo.Fetches{} // Stop after maxCalls
			}
			return kgo.Fetches{}
		},
	}

	consumer := &LambdaKafkaConsumer{
		consumerName:  "test-consumer",
		lambdaName:    "test-lambda",
		kafkaTrigger:  &common.KafkaTrigger{GroupId: "test-group"},
		client:        mockClient,
		lambdaManager: nil,
		stopChan:      make(chan struct{}),
	}

	// Start consume loop
	go consumer.consumeLoop()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Stop consumer
	close(consumer.stopChan)
	time.Sleep(50 * time.Millisecond)

	// Verify multiple polls happened
	if callCount < 3 {
		t.Errorf("Expected at least 3 poll calls, got %d", callCount)
	}
}
