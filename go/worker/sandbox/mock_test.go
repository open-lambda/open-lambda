package sandbox

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MockSandbox is a test implementation of the Sandbox interface
type MockSandbox struct {
	id       string
	paused   bool
	destroyed bool

	// Simulate failures
	pauseError   error
	unpauseError error
	destroyError error

	// Simulate delays
	pauseDelay time.Duration

	// Track calls for verification
	pauseCount   int
	unpauseCount int
	destroyCount int

	mu sync.Mutex
}

func NewMockSandbox(id string) *MockSandbox {
	return &MockSandbox{
		id: id,
	}
}

// Configure mock to fail on specific operations
func (m *MockSandbox) SetPauseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pauseError = err
}

func (m *MockSandbox) SetUnpauseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unpauseError = err
}

func (m *MockSandbox) SetDestroyError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.destroyError = err
}

func (m *MockSandbox) SetPauseDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pauseDelay = delay
}

// Implement Sandbox interface
func (m *MockSandbox) ID() string {
	return m.id
}

func (m *MockSandbox) Pause() error {
	m.mu.Lock()
	pauseCount := m.pauseCount + 1
	m.pauseCount = pauseCount
	destroyed := m.destroyed
	pauseError := m.pauseError
	delay := m.pauseDelay
	m.mu.Unlock()

	// Simulate delay outside of lock to allow timeout detection
	if delay > 0 {
		time.Sleep(delay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if destroyed {
		return fmt.Errorf("sandbox %s already destroyed", m.id)
	}

	if pauseError != nil {
		return pauseError
	}

	m.paused = true
	return nil
}

func (m *MockSandbox) Unpause() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.unpauseCount++

	if m.destroyed {
		return fmt.Errorf("sandbox %s already destroyed", m.id)
	}

	if m.unpauseError != nil {
		return m.unpauseError
	}

	m.paused = false
	return nil
}

func (m *MockSandbox) Destroy(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.destroyCount++
	m.destroyed = true
}

func (m *MockSandbox) DestroyIfPaused(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.paused {
		m.destroyCount++
		m.destroyed = true
	}
}

func (m *MockSandbox) Client() *http.Client {
	return &http.Client{}
}

func (m *MockSandbox) Meta() *SandboxMeta {
	return &SandboxMeta{}
}

func (m *MockSandbox) GetRuntimeLog() string {
	return ""
}

func (m *MockSandbox) GetProxyLog() string {
	return ""
}

func (m *MockSandbox) DebugString() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("MockSandbox{id=%s, paused=%v, destroyed=%v}", m.id, m.paused, m.destroyed)
}

// Private interface methods (not used)
func (m *MockSandbox) fork(dst Sandbox) error {
	return fmt.Errorf("mock does not support fork")
}

func (m *MockSandbox) childExit(child Sandbox) {
	// No-op
}

// Verification helpers
func (m *MockSandbox) WasPaused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pauseCount > 0
}

func (m *MockSandbox) WasUnpaused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.unpauseCount > 0
}

func (m *MockSandbox) WasDestroyed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.destroyed
}

func (m *MockSandbox) PauseCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pauseCount
}

func (m *MockSandbox) UnpauseCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.unpauseCount
}

func (m *MockSandbox) DestroyCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.destroyCount
}

// MockSandboxPool creates mock sandboxes
type MockSandboxPool struct {
	sandboxes []*MockSandbox
	nextID    int

	// Simulate creation failures
	createError error
	createCount int

	mu sync.Mutex
}

func NewMockSandboxPool() *MockSandboxPool {
	return &MockSandboxPool{
		sandboxes: make([]*MockSandbox, 0),
	}
}

func (p *MockSandboxPool) SetCreateError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.createError = err
}

func (p *MockSandboxPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (Sandbox, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.createCount++

	if p.createError != nil {
		return nil, p.createError
	}

	p.nextID++
	sb := NewMockSandbox(fmt.Sprintf("mock-%d", p.nextID))
	p.sandboxes = append(p.sandboxes, sb)
	return sb, nil
}

func (p *MockSandboxPool) Cleanup() {
	// No-op for mock
}

func (p *MockSandboxPool) AddListener(handler SandboxEventFunc) {
	// No-op for mock
}

func (p *MockSandboxPool) DebugString() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return fmt.Sprintf("MockSandboxPool{sandboxes=%d, createCount=%d}", len(p.sandboxes), p.createCount)
}

func (p *MockSandboxPool) CreateCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.createCount
}

func (p *MockSandboxPool) GetAllSandboxes() []*MockSandbox {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]*MockSandbox, len(p.sandboxes))
	copy(result, p.sandboxes)
	return result
}
