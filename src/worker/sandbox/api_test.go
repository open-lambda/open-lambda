package sandbox

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/open-lambda/open-lambda/ol/common"
)

// Mock implementations for testing

type mockSandbox struct {
	id            string
	destroyed     bool
	paused        bool
	meta          *SandboxMeta
	rtType        common.RuntimeType
	destroyReason string
	client        *http.Client
	runtimeLog    string
	proxyLog      string
}

func (m *mockSandbox) ID() string                         { return m.id }
func (m *mockSandbox) Meta() *SandboxMeta                 { return m.meta }
func (m *mockSandbox) GetRuntimeType() common.RuntimeType { return m.rtType }
func (m *mockSandbox) Client() *http.Client               { return m.client }
func (m *mockSandbox) GetRuntimeLog() string              { return m.runtimeLog }
func (m *mockSandbox) GetProxyLog() string                { return m.proxyLog }

func (m *mockSandbox) Destroy(reason string) {
	m.destroyed = true
	m.destroyReason = reason
}

func (m *mockSandbox) DestroyIfPaused(reason string) {
	if m.paused {
		m.Destroy(reason)
	}
}

func (m *mockSandbox) Pause() error {
	if m.destroyed {
		return SandboxDeadError("sandbox is destroyed")
	}
	m.paused = true
	return nil
}

func (m *mockSandbox) Unpause() error {
	if m.destroyed {
		return SandboxDeadError("sandbox is destroyed")
	}
	m.paused = false
	return nil
}

func (m *mockSandbox) fork(_ Sandbox) error {
	if m.destroyed {
		return FORK_FAILED
	}
	return nil
}

func (*mockSandbox) childExit(_ Sandbox) {
	// Mock implementation - no-op
}

func (m *mockSandbox) DebugString() string {
	status := "running"
	if m.destroyed {
		status = "destroyed"
	} else if m.paused {
		status = "paused"
	}
	return fmt.Sprintf("Mock Sandbox %s [%s]", m.id, status)
}

type mockSandboxPool struct {
	sandboxes []Sandbox
	listeners []SandboxEventFunc
	cleaned   bool
}

func (m *mockSandboxPool) Create(_ Sandbox, _ bool, _, _ string, meta *SandboxMeta, rtType common.RuntimeType) (Sandbox, error) {
	if meta == nil {
		meta = &SandboxMeta{MemLimitMB: 128, CPUPercent: 50}
	}

	sb := &mockSandbox{
		id:     fmt.Sprintf("mock-%d", len(m.sandboxes)),
		meta:   meta,
		rtType: rtType,
		client: &http.Client{},
	}
	m.sandboxes = append(m.sandboxes, sb)

	// Notify listeners
	for _, listener := range m.listeners {
		listener(EvCreate, sb)
	}

	return sb, nil
}

func (m *mockSandboxPool) Cleanup() {
	m.cleaned = true
	for _, sb := range m.sandboxes {
		sb.Destroy("cleanup")
	}
}

func (m *mockSandboxPool) AddListener(handler SandboxEventFunc) {
	m.listeners = append(m.listeners, handler)
}

func (m *mockSandboxPool) DebugString() string {
	return fmt.Sprintf("Mock Pool with %d sandboxes", len(m.sandboxes))
}

// Tests

func TestSandboxMeta_String(t *testing.T) {
	tests := []struct {
		name     string
		meta     *SandboxMeta
		expected string
	}{
		{
			name:     "empty meta",
			meta:     &SandboxMeta{},
			expected: "<installs=[], imports=[], mem-limit-mb=0>",
		},
		{
			name: "with installs and imports",
			meta: &SandboxMeta{
				Installs:   []string{"numpy", "pandas"},
				Imports:    []string{"requests", "json"},
				MemLimitMB: 256,
			},
			expected: "<installs=[numpy,pandas], imports=[requests,json], mem-limit-mb=256>",
		},
		{
			name: "single values",
			meta: &SandboxMeta{
				Installs:   []string{"flask"},
				Imports:    []string{"os"},
				MemLimitMB: 128,
			},
			expected: "<installs=[flask], imports=[os], mem-limit-mb=128>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.meta.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSandboxError_Error(t *testing.T) {
	err := SandboxError("test error")
	expected := "test error"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestSandboxDeadError_Error(t *testing.T) {
	err := SandboxDeadError("sandbox is dead")
	expected := "sandbox is dead"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestMockSandbox_BasicOperations(t *testing.T) {
	meta := &SandboxMeta{
		Installs:   []string{"numpy"},
		Imports:    []string{"os"},
		MemLimitMB: 256,
		CPUPercent: 75,
	}

	sb := &mockSandbox{
		id:     "test-sandbox",
		meta:   meta,
		rtType: common.RT_PYTHON,
		client: &http.Client{},
	}

	// Test ID
	if sb.ID() != "test-sandbox" {
		t.Errorf("expected ID 'test-sandbox', got %q", sb.ID())
	}

	// Test Meta
	if sb.Meta() != meta {
		t.Error("Meta() should return the same meta object")
	}

	// Test GetRuntimeType
	if sb.GetRuntimeType() != common.RT_PYTHON {
		t.Errorf("expected runtime type %v, got %v", common.RT_PYTHON, sb.GetRuntimeType())
	}

	// Test Client
	if sb.Client() != sb.client {
		t.Error("Client() should return the same client object")
	}

	// Test initial state
	if sb.destroyed {
		t.Error("sandbox should not be destroyed initially")
	}
	if sb.paused {
		t.Error("sandbox should not be paused initially")
	}
}

func TestMockSandbox_PauseUnpause(t *testing.T) {
	sb := &mockSandbox{id: "test"}

	// Test Pause
	err := sb.Pause()
	if err != nil {
		t.Errorf("unexpected error from Pause(): %v", err)
	}
	if !sb.paused {
		t.Error("sandbox should be paused after Pause()")
	}

	// Test Unpause
	err = sb.Unpause()
	if err != nil {
		t.Errorf("unexpected error from Unpause(): %v", err)
	}
	if sb.paused {
		t.Error("sandbox should not be paused after Unpause()")
	}
}

func TestMockSandbox_Destroy(t *testing.T) {
	sb := &mockSandbox{id: "test"}

	// Test Destroy
	reason := "test destruction"
	sb.Destroy(reason)

	if !sb.destroyed {
		t.Error("sandbox should be destroyed after Destroy()")
	}
	if sb.destroyReason != reason {
		t.Errorf("expected destroy reason %q, got %q", reason, sb.destroyReason)
	}

	// Test operations after destroy return errors
	err := sb.Pause()
	if err == nil {
		t.Error("expected error from Pause() on destroyed sandbox")
	}

	err = sb.Unpause()
	if err == nil {
		t.Error("expected error from Unpause() on destroyed sandbox")
	}
}

func TestMockSandbox_DestroyIfPaused(t *testing.T) {
	tests := []struct {
		name            string
		initiallyPaused bool
		shouldDestroy   bool
	}{
		{"paused sandbox should be destroyed", true, true},
		{"unpaused sandbox should not be destroyed", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := &mockSandbox{id: "test", paused: tt.initiallyPaused}

			reason := "conditional destroy"
			sb.DestroyIfPaused(reason)

			if sb.destroyed != tt.shouldDestroy {
				t.Errorf("expected destroyed=%v, got %v", tt.shouldDestroy, sb.destroyed)
			}

			if tt.shouldDestroy && sb.destroyReason != reason {
				t.Errorf("expected destroy reason %q, got %q", reason, sb.destroyReason)
			}
		})
	}
}

func TestMockSandbox_Fork(t *testing.T) {
	parent := &mockSandbox{id: "parent"}
	child := &mockSandbox{id: "child"}

	// Test successful fork
	err := parent.fork(child)
	if err != nil {
		t.Errorf("unexpected error from fork(): %v", err)
	}

	// Test fork from destroyed sandbox
	parent.Destroy("test")
	err = parent.fork(child)
	if err != FORK_FAILED {
		t.Errorf("expected FORK_FAILED error, got %v", err)
	}
}

func TestMockSandbox_DebugString(t *testing.T) {
	sb := &mockSandbox{id: "test"}

	// Test running state
	debug := sb.DebugString()
	if !strings.Contains(debug, "test") || !strings.Contains(debug, "running") {
		t.Errorf("expected debug string to contain ID and running state, got %q", debug)
	}

	// Test paused state
	sb.paused = true
	debug = sb.DebugString()
	if !strings.Contains(debug, "paused") {
		t.Errorf("expected debug string to contain paused state, got %q", debug)
	}

	// Test destroyed state
	sb.Destroy("test")
	debug = sb.DebugString()
	if !strings.Contains(debug, "destroyed") {
		t.Errorf("expected debug string to contain destroyed state, got %q", debug)
	}
}

func TestMockSandboxPool_Create(t *testing.T) {
	pool := &mockSandboxPool{}

	meta := &SandboxMeta{MemLimitMB: 256}
	sb, err := pool.Create(nil, true, "/code", "/scratch", meta, common.RT_PYTHON)

	if err != nil {
		t.Errorf("unexpected error from Create(): %v", err)
	}
	if sb == nil {
		t.Error("Create() should return a non-nil sandbox")
	}
	if len(pool.sandboxes) != 1 {
		t.Errorf("expected 1 sandbox in pool, got %d", len(pool.sandboxes))
	}
	if sb.Meta() != meta {
		t.Error("sandbox should have the provided meta")
	}
}

func TestMockSandboxPool_AddListener(t *testing.T) {
	pool := &mockSandboxPool{}
	eventReceived := false

	pool.AddListener(func(evType SandboxEventType, _ Sandbox) {
		if evType == EvCreate {
			eventReceived = true
		}
	})

	// Create a sandbox to trigger the event
	_, err := pool.Create(nil, true, "/code", "/scratch", nil, common.RT_PYTHON)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !eventReceived {
		t.Error("listener should have received EvCreate event")
	}
}

func TestMockSandboxPool_Cleanup(t *testing.T) {
	pool := &mockSandboxPool{}

	// Create some sandboxes
	sb1, _ := pool.Create(nil, true, "/code", "/scratch", nil, common.RT_PYTHON)
	sb2, _ := pool.Create(nil, true, "/code", "/scratch", nil, common.RT_PYTHON)

	// Cleanup
	pool.Cleanup()

	if !pool.cleaned {
		t.Error("pool should be marked as cleaned")
	}

	// Check that sandboxes were destroyed
	mockSb1 := sb1.(*mockSandbox)
	mockSb2 := sb2.(*mockSandbox)

	if !mockSb1.destroyed || !mockSb2.destroyed {
		t.Error("all sandboxes should be destroyed after cleanup")
	}
}

func TestMockSandboxPool_DebugString(t *testing.T) {
	pool := &mockSandboxPool{}

	debug := pool.DebugString()
	if !strings.Contains(debug, "0 sandboxes") {
		t.Errorf("expected debug string to mention 0 sandboxes, got %q", debug)
	}

	// Add a sandbox
	pool.Create(nil, true, "/code", "/scratch", nil, common.RT_PYTHON)

	debug = pool.DebugString()
	if !strings.Contains(debug, "1 sandboxes") {
		t.Errorf("expected debug string to mention 1 sandbox, got %q", debug)
	}
}

func TestSandboxEventTypes(t *testing.T) {
	// Test that event types are distinct
	events := []SandboxEventType{
		EvCreate, EvDestroy, EvDestroyIgnored,
		EvPause, EvUnpause, EvFork, EvChildExit,
	}

	seen := make(map[SandboxEventType]bool)
	for _, event := range events {
		if seen[event] {
			t.Errorf("duplicate event type found: %v", event)
		}
		seen[event] = true
	}
}

func TestSandboxConstants(t *testing.T) {
	// Test that error constants are properly defined
	if string(FORK_FAILED) == "" {
		t.Error("FORK_FAILED should not be empty")
	}
	if string(STATUS_UNSUPPORTED) == "" {
		t.Error("STATUS_UNSUPPORTED should not be empty")
	}

	// Test that they implement the error interface
	var err1 error = FORK_FAILED
	var err2 error = STATUS_UNSUPPORTED

	if err1.Error() == "" {
		t.Error("FORK_FAILED should implement error interface")
	}
	if err2.Error() == "" {
		t.Error("STATUS_UNSUPPORTED should implement error interface")
	}
}
