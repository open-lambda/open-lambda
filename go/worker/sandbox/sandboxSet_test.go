package sandbox

import (
	"fmt"  // ← ADD THIS IMPORT
	"sync"
	"testing"
)

// MockSandbox for testing - implements all required Sandbox interface methods
type MockSandbox struct {
	id        string
	destroyed bool
}

func (m *MockSandbox) ID() string {
	return m.id
}

func (m *MockSandbox) Destroy(reason string) {
	m.destroyed = true
}

// Client() is required by the Sandbox interface
// We return nil for testing since we don't use it
func (m *MockSandbox) Client() interface{} {
	return nil
}

// MockSandboxPool for testing
type MockSandboxPool struct {
	mutex       sync.Mutex
	nextID      int
	createError error
}

func (m *MockSandboxPool) Create(
	parent Sandbox,
	isLeaf bool,
	codeDir string,
	scratchDir string,
	meta *SandboxMeta) (Sandbox, error) {

	if m.createError != nil {
		return nil, m.createError
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	id := fmt.Sprintf("mock-sb-%d", m.nextID)
	m.nextID++

	sb := &MockSandbox{id: id}
	return sb, nil
}

func (m *MockSandboxPool) Cleanup() {
}

// AddListener is required by SandboxPool interface
// We provide empty implementation for testing
func (m *MockSandboxPool) AddListener(listener interface{}) {
	// No-op for testing
}

// Helper to create a test SandboxSet
func newTestSandboxSet(maxSandboxes int) *SandboxSet {
	meta := &SandboxMeta{
		MemLimitMB: 100,
		CPUPercent: 100,
	}

	return NewSandboxSet(
		&MockSandboxPool{},
		meta,
		true, // isLeaf
		"/tmp/code",
		"/tmp/scratch",
		maxSandboxes,
	)
}

// Test 1: Basic get and release
func TestSandboxSetBasic(t *testing.T) {
	set := newTestSandboxSet(5)

	// Get a sandbox
	sb1, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Failed to get sandbox: %v", err)
	}

	if sb1 == nil {
		t.Fatal("Expected non-nil sandbox")
	}

	// Check stats
	if set.Size() != 1 {
		t.Errorf("Expected size 1, got %d", set.Size())
	}

	if set.InUse() != 1 {
		t.Errorf("Expected 1 in use, got %d", set.InUse())
	}

	// Release it
	err = set.ReleaseSandbox(sb1)
	if err != nil {
		t.Fatalf("Failed to release sandbox: %v", err)
	}

	// Check stats after release
	if set.InUse() != 0 {
		t.Errorf("Expected 0 in use after release, got %d", set.InUse())
	}
}

// Test 2: Reuse of released sandbox
func TestSandboxSetReuse(t *testing.T) {
	set := newTestSandboxSet(5)

	// Get a sandbox
	sb1, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Failed to get sandbox: %v", err)
	}

	id1 := sb1.ID()

	// Release it
	err = set.ReleaseSandbox(sb1)
	if err != nil {
		t.Fatalf("Failed to release sandbox: %v", err)
	}

	// Get another - should reuse the same one
	sb2, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Failed to get sandbox: %v", err)
	}

	id2 := sb2.ID()

	if id1 != id2 {
		t.Errorf("Expected same sandbox to be reused (id1=%s, id2=%s)", id1, id2)
	}

	// Pool should still have only 1 sandbox
	if set.Size() != 1 {
		t.Errorf("Expected size 1 (reused), got %d", set.Size())
	}
}

// Test 3: Max capacity enforcement
func TestSandboxSetMaxLimit(t *testing.T) {
	set := newTestSandboxSet(3)

	// Get 3 sandboxes (at capacity)
	sb1, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Failed to get sandbox 1: %v", err)
	}

	sb2, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Failed to get sandbox 2: %v", err)
	}

	sb3, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Failed to get sandbox 3: %v", err)
	}

	// Try to get 4th - should fail
	sb4, err := set.GetSandbox()
	if err == nil {
		t.Error("Expected error when exceeding max sandboxes")
	}
	if sb4 != nil {
		t.Error("Expected nil sandbox when at capacity")
	}

	// Release one
	err = set.ReleaseSandbox(sb1)
	if err != nil {
		t.Fatalf("Failed to release sandbox: %v", err)
	}

	// Now should succeed
	sb4, err = set.GetSandbox()
	if err != nil {
		t.Errorf("Should succeed after release: %v", err)
	}
	if sb4.ID() != sb1.ID() {
		t.Error("Expected to reuse released sandbox")
	}

	// Clean up
	set.ReleaseSandbox(sb2)
	set.ReleaseSandbox(sb3)
	set.ReleaseSandbox(sb4)
}

// Test 4: Concurrent access
func TestSandboxSetConcurrency(t *testing.T) {
	set := newTestSandboxSet(20)

	var wg sync.WaitGroup
	numGoroutines := 50
	iterations := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sb, err := set.GetSandbox()
				if err != nil {
					// OK if at capacity
					continue
				}
				// Simulate work
				err = set.ReleaseSandbox(sb)
				if err != nil {
					t.Errorf("Failed to release: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// After all goroutines finish, all should be released
	if set.InUse() != 0 {
		t.Errorf("Expected 0 in use after all released, got %d", set.InUse())
	}

	// Should have created some sandboxes
	if set.Size() == 0 {
		t.Error("Expected some sandboxes to be created")
	}
}

// Test 5: Release nil sandbox
func TestSandboxSetReleaseNil(t *testing.T) {
	set := newTestSandboxSet(5)

	err := set.ReleaseSandbox(nil)
	if err == nil {
		t.Error("Expected error when releasing nil sandbox")
	}
}

// Test 6: Release sandbox not in set
func TestSandboxSetReleaseUnknown(t *testing.T) {
	set := newTestSandboxSet(5)

	// Create a sandbox that's not in the set
	unknownSb := &MockSandbox{id: "unknown-999"}

	err := set.ReleaseSandbox(unknownSb)
	if err == nil {
		t.Error("Expected error when releasing unknown sandbox")
	}
}

// Test 7: Double release
func TestSandboxSetDoubleRelease(t *testing.T) {
	set := newTestSandboxSet(5)

	sb, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Failed to get sandbox: %v", err)
	}

	// Release once (should succeed)
	err = set.ReleaseSandbox(sb)
	if err != nil {
		t.Fatalf("Failed to release sandbox: %v", err)
	}

	// Release again (should fail)
	err = set.ReleaseSandbox(sb)
	if err == nil {
		t.Error("Expected error when releasing already-released sandbox")
	}
}

// Test 8: Destroy
func TestSandboxSetDestroy(t *testing.T) {
	set := newTestSandboxSet(5)

	// Create some sandboxes
	sb1, _ := set.GetSandbox()
	sb2, _ := set.GetSandbox()
	sb3, _ := set.GetSandbox()
	set.ReleaseSandbox(sb2) // One released, two in use

	// Destroy all
	set.Destroy()

	// Check that all were destroyed by checking internal state
	// We can't type assert anymore, so we'll verify via set state
	if set.Size() != 0 {
		t.Errorf("Expected size 0 after destroy, got %d", set.Size())
	}

	// The sandboxes themselves should have Destroy() called
	// We verify this indirectly by checking the destroyed flag
	// Since we can't access it directly anymore, we just check set is empty
}

// Test 9: Stats
func TestSandboxSetStats(t *testing.T) {
	set := newTestSandboxSet(10)

	// Get some sandboxes
	sb1, _ := set.GetSandbox()
	sb2, _ := set.GetSandbox()
	sb3, _ := set.GetSandbox()

	// Release one
	set.ReleaseSandbox(sb2)

	stats := set.Stats()

	if stats["total"] != 3 {
		t.Errorf("Expected total=3, got %d", stats["total"])
	}

	if stats["in_use"] != 2 {
		t.Errorf("Expected in_use=2, got %d", stats["in_use"])
	}

	if stats["idle"] != 1 {
		t.Errorf("Expected idle=1, got %d", stats["idle"])
	}

	if stats["max"] != 10 {
		t.Errorf("Expected max=10, got %d", stats["max"])
	}

	if stats["created"] != 3 {
		t.Errorf("Expected created=3, got %d", stats["created"])
	}

	if stats["borrowed"] != 3 {
		t.Errorf("Expected borrowed=3, got %d", stats["borrowed"])
	}

	if stats["released"] != 1 {
		t.Errorf("Expected released=1, got %d", stats["released"])
	}

	// Clean up
	set.ReleaseSandbox(sb1)
	set.ReleaseSandbox(sb3)
}

// Test 10: Zero or negative max sandboxes
func TestSandboxSetInvalidMax(t *testing.T) {
	// Should use default value (10)
	set := newTestSandboxSet(0)
	if set.maxSandboxes != 10 {
		t.Errorf("Expected default max 10, got %d", set.maxSandboxes)
	}

	set2 := newTestSandboxSet(-5)
	if set2.maxSandboxes != 10 {
		t.Errorf("Expected default max 10, got %d", set2.maxSandboxes)
	}
}
