package sandbox

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
)

// MockSandbox is a test double for Sandbox.
// Exported fields control error injection; state fields track lifecycle.
type MockSandbox struct {
	mu        sync.Mutex
	id        string
	paused    bool
	destroyed bool

	// Set these before calling Get/Put to inject errors.
	PauseErr   error
	UnpauseErr error
}

var mockIDCounter int64

// NewMockSandbox creates a MockSandbox with the given ID.
func NewMockSandbox(id string) *MockSandbox {
	return &MockSandbox{id: id, paused: true}
}

func (m *MockSandbox) ID() string { return m.id }

func (m *MockSandbox) Destroy(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.destroyed = true
}

func (m *MockSandbox) DestroyIfPaused(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paused {
		m.destroyed = true
	}
}

func (m *MockSandbox) Pause() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.PauseErr != nil {
		return m.PauseErr
	}
	m.paused = true
	return nil
}

func (m *MockSandbox) Unpause() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UnpauseErr != nil {
		return m.UnpauseErr
	}
	m.paused = false
	return nil
}

func (m *MockSandbox) Client() *http.Client   { return nil }
func (m *MockSandbox) Meta() *SandboxMeta      { return nil }
func (m *MockSandbox) GetRuntimeLog() string   { return "" }
func (m *MockSandbox) GetProxyLog() string     { return "" }
func (m *MockSandbox) DebugString() string     { return fmt.Sprintf("mock:%s", m.id) }
func (m *MockSandbox) fork(dst Sandbox) error  { return nil }
func (m *MockSandbox) childExit(child Sandbox) {}

// IsDestroyed returns whether Destroy has been called.
func (m *MockSandbox) IsDestroyed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.destroyed
}

// IsPaused returns the current pause state.
func (m *MockSandbox) IsPaused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paused
}

// MockSandboxPool is a test double for SandboxPool.
// It creates MockSandbox instances with auto-incremented IDs.
type MockSandboxPool struct {
	mu      sync.Mutex
	Created []*MockSandbox

	// Set before calling Get to make pool.Create fail.
	CreateErr error
}

func (p *MockSandboxPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (Sandbox, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.CreateErr != nil {
		return nil, p.CreateErr
	}
	id := fmt.Sprintf("mock-%d", atomic.AddInt64(&mockIDCounter, 1))
	sb := NewMockSandbox(id)
	sb.paused = false // Pool.Create returns unpaused sandboxes
	p.Created = append(p.Created, sb)
	return sb, nil
}

func (p *MockSandboxPool) Cleanup()                            {}
func (p *MockSandboxPool) AddListener(handler SandboxEventFunc) {}
func (p *MockSandboxPool) DebugString() string                  { return "mock-pool" }

// CreatedSandboxes returns a snapshot of all sandboxes created by this pool.
func (p *MockSandboxPool) CreatedSandboxes() []*MockSandbox {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]*MockSandbox, len(p.Created))
	copy(out, p.Created)
	return out
}
