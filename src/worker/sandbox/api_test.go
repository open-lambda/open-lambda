package sandbox

import (
	"net/http"
	"testing"

	"github.com/open-lambda/open-lambda/ol/common"
)

// MockSandbox implements the Sandbox interface for testing
type MockSandbox struct {
	id          string
	meta        *SandboxMeta
	destroyed   bool
	paused      bool
	client      *http.Client
	runtimeType common.RuntimeType
}

func NewMockSandbox(id string, meta *SandboxMeta) *MockSandbox {
	return &MockSandbox{
		id:          id,
		meta:        meta,
		destroyed:   false,
		paused:      false,
		client:      &http.Client{},
		runtimeType: common.RT_PYTHON,
	}
}

func (m *MockSandbox) ID() string {
	return m.id
}

func (m *MockSandbox) Destroy(reason string) {
	m.destroyed = true
}

func (m *MockSandbox) DestroyIfPaused(reason string) {
	if m.paused {
		m.destroyed = true
	}
}

func (m *MockSandbox) Pause() error {
	if m.destroyed {
		return SandboxDeadError("sandbox is destroyed")
	}
	m.paused = true
	return nil
}

func (m *MockSandbox) Unpause() error {
	if m.destroyed {
		return SandboxDeadError("sandbox is destroyed")
	}
	m.paused = false
	return nil
}

func (m *MockSandbox) Client() *http.Client {
	return m.client
}

func (m *MockSandbox) Meta() *SandboxMeta {
	return m.meta
}

func (m *MockSandbox) GetRuntimeLog() string {
	return "mock runtime log"
}

func (m *MockSandbox) GetProxyLog() string {
	return "mock proxy log"
}

func (m *MockSandbox) DebugString() string {
	return "MockSandbox{id: " + m.id + "}"
}

func (m *MockSandbox) fork(dst Sandbox) error {
	return nil
}

func (m *MockSandbox) childExit(child Sandbox) {
	// No-op for mock
}

func (m *MockSandbox) GetRuntimeType() common.RuntimeType {
	return m.runtimeType
}

// MockSandboxPool implements the SandboxPool interface for testing
type MockSandboxPool struct {
	sandboxes []Sandbox
	listeners []SandboxEventFunc
}

func NewMockSandboxPool() *MockSandboxPool {
	return &MockSandboxPool{
		sandboxes: make([]Sandbox, 0),
		listeners: make([]SandboxEventFunc, 0),
	}
}

func (p *MockSandboxPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta, rtType common.RuntimeType) (Sandbox, error) {
	if meta == nil {
		meta = &SandboxMeta{
			Installs:   []string{},
			Imports:    []string{},
			MemLimitMB: 128,
			CPUPercent: 100,
		}
	}

	sandbox := NewMockSandbox("mock-sandbox-"+string(rune(len(p.sandboxes)+1)), meta)
	sandbox.runtimeType = rtType
	p.sandboxes = append(p.sandboxes, sandbox)

	// Notify listeners
	for _, listener := range p.listeners {
		listener(EvCreate, sandbox)
	}

	return sandbox, nil
}

func (p *MockSandboxPool) Cleanup() {
	for _, sb := range p.sandboxes {
		sb.Destroy("cleanup")
	}
	p.sandboxes = make([]Sandbox, 0)
}

func (p *MockSandboxPool) AddListener(handler SandboxEventFunc) {
	p.listeners = append(p.listeners, handler)
}

func (p *MockSandboxPool) DebugString() string {
	return "MockSandboxPool with " + string(rune(len(p.sandboxes))) + " sandboxes"
}

// Test SandboxMeta structure
func TestSandboxMeta(t *testing.T) {
	meta := &SandboxMeta{
		Installs:   []string{"numpy", "pandas"},
		Imports:    []string{"requests", "json"},
		MemLimitMB: 256,
		CPUPercent: 50,
	}

	if len(meta.Installs) != 2 {
		t.Errorf("Expected 2 installs, got %d", len(meta.Installs))
	}

	if meta.Installs[0] != "numpy" {
		t.Errorf("Expected first install to be 'numpy', got %s", meta.Installs[0])
	}

	if len(meta.Imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(meta.Imports))
	}

	if meta.MemLimitMB != 256 {
		t.Errorf("Expected memory limit 256, got %d", meta.MemLimitMB)
	}

	if meta.CPUPercent != 50 {
		t.Errorf("Expected CPU percent 50, got %d", meta.CPUPercent)
	}
}

// Test Sandbox interface implementation
func TestSandboxInterface(t *testing.T) {
	meta := &SandboxMeta{
		Installs:   []string{"test-package"},
		Imports:    []string{"test-import"},
		MemLimitMB: 128,
		CPUPercent: 100,
	}

	sandbox := NewMockSandbox("test-sandbox", meta)

	// Test ID
	if sandbox.ID() != "test-sandbox" {
		t.Errorf("Expected ID 'test-sandbox', got %s", sandbox.ID())
	}

	// Test Meta
	retrievedMeta := sandbox.Meta()
	if retrievedMeta.MemLimitMB != 128 {
		t.Errorf("Expected memory limit 128, got %d", retrievedMeta.MemLimitMB)
	}

	// Test Client
	client := sandbox.Client()
	if client == nil {
		t.Error("Expected non-nil HTTP client")
	}

	// Test Pause/Unpause
	err := sandbox.Pause()
	if err != nil {
		t.Errorf("Expected no error on pause, got %v", err)
	}

	err = sandbox.Unpause()
	if err != nil {
		t.Errorf("Expected no error on unpause, got %v", err)
	}

	// Test DestroyIfPaused (should not destroy when not paused)
	sandbox.DestroyIfPaused("test")
	if sandbox.destroyed {
		t.Error("Expected sandbox not to be destroyed when not paused")
	}

	// Test DestroyIfPaused (should destroy when paused)
	sandbox.Pause()
	sandbox.DestroyIfPaused("test")
	if !sandbox.destroyed {
		t.Error("Expected sandbox to be destroyed when paused")
	}

	// Test operations on destroyed sandbox
	err = sandbox.Pause()
	if err == nil {
		t.Error("Expected error when pausing destroyed sandbox")
	}

	// Test GetRuntimeType
	rtType := sandbox.GetRuntimeType()
	if rtType != common.RT_PYTHON {
		t.Errorf("Expected runtime type RT_PYTHON, got %v", rtType)
	}

	// Test logs
	runtimeLog := sandbox.GetRuntimeLog()
	if runtimeLog == "" {
		t.Error("Expected non-empty runtime log")
	}

	proxyLog := sandbox.GetProxyLog()
	if proxyLog == "" {
		t.Error("Expected non-empty proxy log")
	}

	// Test DebugString
	debugStr := sandbox.DebugString()
	if debugStr == "" {
		t.Error("Expected non-empty debug string")
	}
}

// Test SandboxPool interface implementation
func TestSandboxPoolInterface(t *testing.T) {
	pool := NewMockSandboxPool()

	// Test initial state
	debugStr := pool.DebugString()
	if debugStr == "" {
		t.Error("Expected non-empty debug string")
	}

	// Test event listener
	var receivedEvents []SandboxEventType
	listener := func(evType SandboxEventType, sb Sandbox) {
		receivedEvents = append(receivedEvents, evType)
	}
	pool.AddListener(listener)

	// Test Create
	meta := &SandboxMeta{
		Installs:   []string{"test"},
		Imports:    []string{"import"},
		MemLimitMB: 256,
		CPUPercent: 100,
	}

	sandbox, err := pool.Create(nil, true, "/code", "/scratch", meta, common.RT_PYTHON)
	if err != nil {
		t.Errorf("Expected no error on create, got %v", err)
	}

	if sandbox == nil {
		t.Error("Expected non-nil sandbox")
	}

	if len(receivedEvents) != 1 || receivedEvents[0] != EvCreate {
		t.Errorf("Expected EvCreate event, got %v", receivedEvents)
	}

	// Test Create with nil meta (should use defaults)
	sandbox2, err := pool.Create(nil, true, "/code", "/scratch", nil, common.RT_NATIVE)
	if err != nil {
		t.Errorf("Expected no error on create with nil meta, got %v", err)
	}

	if sandbox2.Meta().MemLimitMB != 128 {
		t.Errorf("Expected default memory limit 128, got %d", sandbox2.Meta().MemLimitMB)
	}

	// Test Cleanup
	pool.Cleanup()
	// After cleanup, sandboxes should be destroyed
	// Note: In a real implementation, we'd check that resources are cleaned up
}

// Test SandboxError types
func TestSandboxErrors(t *testing.T) {
	err := SandboxError("test error")
	if string(err) != "test error" {
		t.Errorf("Expected 'test error', got %s", string(err))
	}

	deadErr := SandboxDeadError("sandbox dead")
	if string(deadErr) != "sandbox dead" {
		t.Errorf("Expected 'sandbox dead', got %s", string(deadErr))
	}

	// Test constant errors
	if string(FORK_FAILED) != "Fork from parent Sandbox failed" {
		t.Errorf("Unexpected FORK_FAILED message: %s", string(FORK_FAILED))
	}

	if string(STATUS_UNSUPPORTED) != "Argument to Status(...) unsupported by this Sandbox" {
		t.Errorf("Unexpected STATUS_UNSUPPORTED message: %s", string(STATUS_UNSUPPORTED))
	}
}

// Test SandboxEventType constants
func TestSandboxEventTypes(t *testing.T) {
	events := []SandboxEventType{
		EvCreate,
		EvDestroy,
		EvDestroyIgnored,
		EvPause,
		EvUnpause,
		EvFork,
		EvChildExit,
	}

	// Ensure all event types are distinct
	eventSet := make(map[SandboxEventType]bool)
	for _, event := range events {
		if eventSet[event] {
			t.Errorf("Duplicate event type: %v", event)
		}
		eventSet[event] = true
	}

	if len(eventSet) != len(events) {
		t.Errorf("Expected %d unique events, got %d", len(events), len(eventSet))
	}
}

// Test edge cases and error conditions
func TestSandboxEdgeCases(t *testing.T) {
	// Test creating sandbox with parent (not supported in DockerPool)
	pool := NewMockSandboxPool()

	// Create a parent sandbox first
	parent, err := pool.Create(nil, false, "/code", "/scratch", nil, common.RT_PYTHON)
	if err != nil {
		t.Errorf("Failed to create parent sandbox: %v", err)
	}

	// Try to create child from parent (our mock allows this, but real DockerPool might not)
	child, err := pool.Create(parent, true, "/code", "/scratch", nil, common.RT_PYTHON)
	if err != nil {
		t.Errorf("Failed to create child sandbox: %v", err)
	}

	if child == nil {
		t.Error("Expected non-nil child sandbox")
	}

	// Test multiple listeners
	var events1, events2 []SandboxEventType
	pool.AddListener(func(evType SandboxEventType, sb Sandbox) {
		events1 = append(events1, evType)
	})
	pool.AddListener(func(evType SandboxEventType, sb Sandbox) {
		events2 = append(events2, evType)
	})

	// Create another sandbox to trigger events
	_, err = pool.Create(nil, true, "/code", "/scratch", nil, common.RT_PYTHON)
	if err != nil {
		t.Errorf("Failed to create sandbox: %v", err)
	}

	// Both listeners should have received the event
	if len(events1) == 0 || len(events2) == 0 {
		t.Error("Expected both listeners to receive events")
	}
}

// Test SandboxMeta with empty values
func TestSandboxMetaEmpty(t *testing.T) {
	meta := &SandboxMeta{}

	if meta.Installs == nil {
		meta.Installs = []string{}
	}
	if meta.Imports == nil {
		meta.Imports = []string{}
	}

	if len(meta.Installs) != 0 {
		t.Errorf("Expected 0 installs, got %d", len(meta.Installs))
	}

	if len(meta.Imports) != 0 {
		t.Errorf("Expected 0 imports, got %d", len(meta.Imports))
	}

	if meta.MemLimitMB != 0 {
		t.Errorf("Expected 0 memory limit, got %d", meta.MemLimitMB)
	}

	if meta.CPUPercent != 0 {
		t.Errorf("Expected 0 CPU percent, got %d", meta.CPUPercent)
	}
}
