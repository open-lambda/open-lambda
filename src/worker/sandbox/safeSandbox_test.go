package sandbox

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/open-lambda/open-lambda/ol/common"
)

// Mock sandbox that can simulate errors
type errorMockSandbox struct {
	*mockSandbox
	pauseError   error
	unpauseError error
	forkError    error
}

func (e *errorMockSandbox) Pause() error {
	if e.pauseError != nil {
		return e.pauseError
	}
	return e.mockSandbox.Pause()
}

func (e *errorMockSandbox) Unpause() error {
	if e.unpauseError != nil {
		return e.unpauseError
	}
	return e.mockSandbox.Unpause()
}

func (e *errorMockSandbox) fork(dst Sandbox) error {
	if e.forkError != nil {
		return e.forkError
	}
	return e.mockSandbox.fork(dst)
}

func init() {
	// Initialize configuration for tests
	if common.Conf == nil {
		common.Conf = &common.Config{
			Trace: common.TraceConfig{
				Latency: false,
			},
			Limits: common.LimitsConfig{
				Mem_mb:      128,
				CPU_percent: 50,
			},
		}
	}
}

func TestNewSafeSandbox(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)

	if safe == nil {
		t.Error("newSafeSandbox should return non-nil sandbox")
	}
	if safe.Sandbox != inner {
		t.Error("safeSandbox should wrap the inner sandbox")
	}
	if safe.paused {
		t.Error("safeSandbox should not be paused initially")
	}
	if safe.dead != nil {
		t.Error("safeSandbox should not be dead initially")
	}
}

func TestSafeSandbox_StartNotifyingListeners(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)

	eventReceived := false
	var receivedEventType SandboxEventType
	var receivedSandbox Sandbox

	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			eventReceived = true
			receivedEventType = evType
			receivedSandbox = sb
		},
	}

	safe.startNotifyingListeners(handlers)

	if !eventReceived {
		t.Error("EvCreate event should be sent when starting to notify listeners")
	}
	if receivedEventType != EvCreate {
		t.Errorf("expected EvCreate, got %v", receivedEventType)
	}
	if receivedSandbox != safe {
		t.Error("event should include the safe sandbox as the subject")
	}
}

func TestSafeSandbox_Destroy(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)

	eventReceived := false
	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvDestroy {
				eventReceived = true
			}
		},
	}
	safe.startNotifyingListeners(handlers)
	eventReceived = false // Reset after EvCreate

	reason := "test destroy"
	safe.Destroy(reason)

	// Check that inner sandbox was destroyed
	mockInner := inner
	if !mockInner.destroyed {
		t.Error("inner sandbox should be destroyed")
	}
	if mockInner.destroyReason != reason {
		t.Errorf("expected destroy reason %q, got %q", reason, mockInner.destroyReason)
	}

	// Check that safe sandbox is marked as dead
	if safe.dead == nil {
		t.Error("safeSandbox should be marked as dead")
	}

	// Check that event was sent
	if !eventReceived {
		t.Error("EvDestroy event should be sent")
	}

	// Test that subsequent calls are no-ops
	safe.Destroy("second call")
	// Should not panic or cause issues
}

func TestSafeSandbox_DestroyIfPaused(t *testing.T) {
	tests := []struct {
		name            string
		initiallyPaused bool
		expectedEvent   SandboxEventType
		shouldDestroy   bool
	}{
		{"destroy paused sandbox", true, EvDestroy, true},
		{"ignore unpaused sandbox", false, EvDestroyIgnored, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := &mockSandbox{id: "test", paused: tt.initiallyPaused}
			safe := newSafeSandbox(inner)

			var receivedEvent SandboxEventType
			handlers := []SandboxEventFunc{
				func(evType SandboxEventType, sb Sandbox) {
					if evType != EvCreate {
						receivedEvent = evType
					}
				},
			}
			safe.startNotifyingListeners(handlers)
			safe.paused = tt.initiallyPaused

			safe.DestroyIfPaused("test")

			if receivedEvent != tt.expectedEvent {
				t.Errorf("expected event %v, got %v", tt.expectedEvent, receivedEvent)
			}

			mockInner := inner
			if mockInner.destroyed != tt.shouldDestroy {
				t.Errorf("expected destroyed=%v, got %v", tt.shouldDestroy, mockInner.destroyed)
			}
		})
	}
}

func TestSafeSandbox_Pause(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)

	var receivedEvent SandboxEventType
	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvPause {
				receivedEvent = evType
			}
		},
	}
	safe.startNotifyingListeners(handlers)

	// Test successful pause
	err := safe.Pause()
	if err != nil {
		t.Errorf("unexpected error from Pause(): %v", err)
	}
	if !safe.paused {
		t.Error("safeSandbox should be marked as paused")
	}
	if receivedEvent != EvPause {
		t.Error("EvPause event should be sent")
	}

	// Test pause when already paused (should be no-op)
	receivedEvent = 0 // Reset
	err = safe.Pause()
	if err != nil {
		t.Errorf("unexpected error from Pause() when already paused: %v", err)
	}
	if receivedEvent == EvPause {
		t.Error("EvPause event should not be sent when already paused")
	}
}

func TestSafeSandbox_Unpause(t *testing.T) {
	inner := &mockSandbox{id: "test", paused: true}
	safe := newSafeSandbox(inner)
	safe.paused = true

	var receivedEvent SandboxEventType
	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvUnpause {
				receivedEvent = evType
			}
		},
	}
	safe.startNotifyingListeners(handlers)

	// Test successful unpause
	err := safe.Unpause()
	if err != nil {
		t.Errorf("unexpected error from Unpause(): %v", err)
	}
	if safe.paused {
		t.Error("safeSandbox should not be marked as paused")
	}
	if receivedEvent != EvUnpause {
		t.Error("EvUnpause event should be sent")
	}

	// Test unpause when already unpaused (should be no-op)
	receivedEvent = 0 // Reset
	err = safe.Unpause()
	if err != nil {
		t.Errorf("unexpected error from Unpause() when already unpaused: %v", err)
	}
	if receivedEvent == EvUnpause {
		t.Error("EvUnpause event should not be sent when already unpaused")
	}
}

func TestSafeSandbox_ErrorHandling(t *testing.T) {
	testError := errors.New("test error")
	inner := &errorMockSandbox{
		mockSandbox: &mockSandbox{id: "test"},
		pauseError:  testError,
	}
	safe := newSafeSandbox(inner)

	var destroyEventReceived bool
	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvDestroy {
				destroyEventReceived = true
			}
		},
	}
	safe.startNotifyingListeners(handlers)

	// Pause should fail and cause sandbox to be destroyed
	err := safe.Pause()
	if err == nil {
		t.Error("expected error from Pause()")
	}
	if err != testError {
		t.Errorf("expected original error, got %v", err)
	}

	// Check that sandbox was automatically destroyed
	if !destroyEventReceived {
		t.Error("sandbox should be automatically destroyed on error")
	}
	if safe.dead == nil {
		t.Error("safeSandbox should be marked as dead")
	}

	// Subsequent operations should return dead error
	err = safe.Pause()
	if _, ok := err.(SandboxDeadError); !ok {
		t.Errorf("expected SandboxDeadError, got %T", err)
	}
}

func TestSafeSandbox_Fork(t *testing.T) {
	inner := &mockSandbox{id: "parent"}
	safe := newSafeSandbox(inner)
	child := &mockSandbox{id: "child"}

	var receivedEvent SandboxEventType
	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvFork {
				receivedEvent = evType
			}
		},
	}
	safe.startNotifyingListeners(handlers)

	// Test successful fork
	err := safe.fork(child)
	if err != nil {
		t.Errorf("unexpected error from fork(): %v", err)
	}
	if receivedEvent != EvFork {
		t.Error("EvFork event should be sent")
	}

	// Test fork from dead sandbox
	safe.Destroy("test")
	err = safe.fork(child)
	if _, ok := err.(SandboxDeadError); !ok {
		t.Errorf("expected SandboxDeadError, got %T", err)
	}
}

func TestSafeSandbox_ChildExit(t *testing.T) {
	inner := &mockSandbox{id: "parent"}
	safe := newSafeSandbox(inner)
	child := &mockSandbox{id: "child"}

	var receivedEvent SandboxEventType
	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvChildExit {
				receivedEvent = evType
			}
		},
	}
	safe.startNotifyingListeners(handlers)

	// Test child exit on live sandbox
	safe.childExit(child)
	if receivedEvent != EvChildExit {
		t.Error("EvChildExit event should be sent")
	}

	// Test child exit on dead sandbox (should not send event)
	safe.Destroy("test")
	receivedEvent = 0 // Reset
	safe.childExit(child)
	if receivedEvent == EvChildExit {
		t.Error("EvChildExit event should not be sent for dead sandbox")
	}
}

func TestSafeSandbox_Client(t *testing.T) {
	client := &http.Client{}
	inner := &mockSandbox{id: "test", client: client}
	safe := newSafeSandbox(inner)

	if safe.Client() != client {
		t.Error("Client() should return the inner sandbox's client")
	}
}

func TestSafeSandbox_DebugString(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)

	// Test live sandbox
	debug := safe.DebugString()
	if !strings.Contains(debug, "test") {
		t.Errorf("debug string should contain sandbox ID, got %q", debug)
	}

	// Test dead sandbox
	safe.Destroy("test")
	debug = safe.DebugString()
	if !strings.Contains(debug, "DEAD") {
		t.Errorf("debug string should indicate dead status, got %q", debug)
	}
}

func TestSafeSandbox_ConcurrentAccess(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)
	safe.startNotifyingListeners(nil)

	// Test concurrent pause/unpause operations
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Launch goroutines that pause
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			safe.Pause()
		}()
	}

	// Launch goroutines that unpause
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			safe.Unpause()
		}()
	}

	wg.Wait()

	// Should not panic and sandbox should be in a consistent state
	if safe.dead != nil {
		t.Error("sandbox should not be dead after concurrent operations")
	}
}

func TestSafeSandbox_EventSequence(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)

	var events []SandboxEventType
	var mu sync.Mutex

	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			mu.Lock()
			events = append(events, evType)
			mu.Unlock()
		},
	}

	safe.startNotifyingListeners(handlers)

	// Perform a sequence of operations
	safe.Pause()
	safe.Unpause()
	child := &mockSandbox{id: "child"}
	safe.fork(child)
	safe.childExit(child)
	safe.Destroy("test")

	mu.Lock()
	expectedEvents := []SandboxEventType{
		EvCreate, EvPause, EvUnpause, EvFork, EvChildExit, EvDestroy,
	}
	mu.Unlock()

	if len(events) != len(expectedEvents) {
		t.Errorf("expected %d events, got %d", len(expectedEvents), len(events))
	}

	for i, expected := range expectedEvents {
		if i < len(events) && events[i] != expected {
			t.Errorf("event %d: expected %v, got %v", i, expected, events[i])
		}
	}
}

func TestSafeSandbox_MultipleListeners(t *testing.T) {
	inner := &mockSandbox{id: "test"}
	safe := newSafeSandbox(inner)

	listener1Called := false
	listener2Called := false

	handlers := []SandboxEventFunc{
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvCreate {
				listener1Called = true
			}
		},
		func(evType SandboxEventType, sb Sandbox) {
			if evType == EvCreate {
				listener2Called = true
			}
		},
	}

	safe.startNotifyingListeners(handlers)

	if !listener1Called {
		t.Error("first listener should be called")
	}
	if !listener2Called {
		t.Error("second listener should be called")
	}
}