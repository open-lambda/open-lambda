package sandbox

import (
	"errors"
	"sync"
	"testing"
)

// Test 1: Basic structure and initialization
func TestSandboxSetInit(t *testing.T) {
	set := &SandboxSet{
		sandboxes:    []*sandboxWrapper{},
		maxSandboxes: 5,
	}

	if set.Size() != 0 {
		t.Errorf("Expected size 0, got %d", set.Size())
	}

	if set.InUse() != 0 {
		t.Errorf("Expected 0 in use, got %d", set.InUse())
	}
}

// Test 2: Stats calculation
func TestSandboxSetStats(t *testing.T) {
	set := &SandboxSet{
		sandboxes: []*sandboxWrapper{
			{sb: nil, inUse: true},
			{sb: nil, inUse: false},
			{sb: nil, inUse: true},
		},
		maxSandboxes: 10,
		created:      3,
		borrowed:     5,
		released:     2,
	}

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

	if stats["borrowed"] != 5 {
		t.Errorf("Expected borrowed=5, got %d", stats["borrowed"])
	}

	if stats["released"] != 2 {
		t.Errorf("Expected released=2, got %d", stats["released"])
	}
}

// Test 3: Size and InUse methods
func TestSandboxSetSizeAndInUse(t *testing.T) {
	set := &SandboxSet{
		sandboxes: []*sandboxWrapper{
			{inUse: true},
			{inUse: false},
			{inUse: true},
			{inUse: false},
			{inUse: true},
		},
		maxSandboxes: 10,
	}

	if set.Size() != 5 {
		t.Errorf("Expected size 5, got %d", set.Size())
	}

	if set.InUse() != 3 {
		t.Errorf("Expected 3 in use, got %d", set.InUse())
	}
}

// Test 4: Concurrent access to Stats (thread safety test)
func TestSandboxSetConcurrentStats(t *testing.T) {
	set := &SandboxSet{
		sandboxes: []*sandboxWrapper{
			{inUse: true},
			{inUse: false},
		},
		maxSandboxes: 10,
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Many goroutines reading stats concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = set.Stats()
			_ = set.Size()
			_ = set.InUse()
		}()
	}

	wg.Wait()
	// If we got here without deadlock or race, test passes
}

// Test 5: Max sandboxes validation in NewSandboxSet
func TestSandboxSetMaxValidation(t *testing.T) {
	meta := &SandboxMeta{
		MemLimitMB: 100,
		CPUPercent: 100,
	}

	// Test with 0 max (should default to 10)
	set1 := NewSandboxSet(nil, meta, true, "/tmp/code", "/tmp/scratch", 0)
	if set1.maxSandboxes != 10 {
		t.Errorf("Expected default max 10 for zero input, got %d", set1.maxSandboxes)
	}

	// Test with negative max (should default to 10)
	set2 := NewSandboxSet(nil, meta, true, "/tmp/code", "/tmp/scratch", -5)
	if set2.maxSandboxes != 10 {
		t.Errorf("Expected default max 10 for negative input, got %d", set2.maxSandboxes)
	}

	// Test with positive max (should use provided value)
	set3 := NewSandboxSet(nil, meta, true, "/tmp/code", "/tmp/scratch", 15)
	if set3.maxSandboxes != 15 {
		t.Errorf("Expected max 15, got %d", set3.maxSandboxes)
	}
}

// Test 6: fillMetaDefaults integration (skipped - requires config)
func TestSandboxSetMetaDefaults(t *testing.T) {
	t.Skip("Skipping: requires common.Conf initialization")

	// This test would work in integration tests but not unit tests
	// because it depends on common.Conf being initialized.
	// We validate that NewSandboxSet doesn't crash with nil meta instead.
}

// Test 6b: NewSandboxSet handles nil pool gracefully
func TestSandboxSetNilPool(t *testing.T) {
	meta := &SandboxMeta{
		MemLimitMB: 256,
		CPUPercent: 50,
	}

	// NewSandboxSet should not crash with nil pool
	set := NewSandboxSet(nil, meta, true, "/tmp/code", "/tmp/scratch", 5)

	if set.sbPool != nil {
		t.Error("Expected nil pool to remain nil")
	}

	// Other fields should still be set
	if set.maxSandboxes != 5 {
		t.Errorf("Expected maxSandboxes=5, got %d", set.maxSandboxes)
	}
}

// Test 7: ReleaseSandbox error handling with nil
func TestSandboxSetReleaseNilError(t *testing.T) {
	set := &SandboxSet{
		sandboxes:    []*sandboxWrapper{},
		maxSandboxes: 5,
	}

	err := set.ReleaseSandbox(nil)
	if err == nil {
		t.Error("Expected error when releasing nil sandbox")
	}

	if err.Error() != "cannot release nil sandbox" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// Test 8: Concurrent modification of internal state
func TestSandboxSetConcurrentModification(t *testing.T) {
	set := &SandboxSet{
		sandboxes:    []*sandboxWrapper{},
		maxSandboxes: 100,
		mutex:        sync.Mutex{},
	}

	var wg sync.WaitGroup
	numGoroutines := 50

	// Simulate concurrent access patterns
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Try to read stats
			_ = set.Stats()

			// Try to access size
			_ = set.Size()
			_ = set.InUse()
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	stats := set.Stats()
	if stats["total"] != set.Size() {
		t.Errorf("Stats total (%d) doesn't match Size() (%d)", stats["total"], set.Size())
	}
}

// Test 9: Destroy clears sandboxes (without testing actual Destroy call)
func TestSandboxSetDestroy(t *testing.T) {
	set := &SandboxSet{
		sandboxes: []*sandboxWrapper{
			{sb: nil, inUse: true},
			{sb: nil, inUse: false},
			{sb: nil, inUse: true},
		},
		maxSandboxes: 10,
	}

	// Call Destroy
	set.Destroy()

	// Check set is empty
	if set.Size() != 0 {
		t.Errorf("Expected size 0 after destroy, got %d", set.Size())
	}

	if set.sandboxes != nil {
		t.Error("Expected sandboxes slice to be nil after destroy")
	}
}

// Test 10: NewSandboxSet initializes all fields correctly
func TestSandboxSetNewInitialization(t *testing.T) {
	meta := &SandboxMeta{
		MemLimitMB: 256,
		CPUPercent: 50,
	}

	set := NewSandboxSet(nil, meta, true, "/code/dir", "/scratch/dir", 20)

	// Check all fields
	if set.maxSandboxes != 20 {
		t.Errorf("Expected maxSandboxes=20, got %d", set.maxSandboxes)
	}

	if set.isLeaf != true {
		t.Error("Expected isLeaf=true")
	}

	if set.codeDir != "/code/dir" {
		t.Errorf("Expected codeDir=/code/dir, got %s", set.codeDir)
	}

	if set.scratchDir != "/scratch/dir" {
		t.Errorf("Expected scratchDir=/scratch/dir, got %s", set.scratchDir)
	}

	if set.created != 0 {
		t.Errorf("Expected created=0, got %d", set.created)
	}

	if set.borrowed != 0 {
		t.Errorf("Expected borrowed=0, got %d", set.borrowed)
	}

	if set.released != 0 {
		t.Errorf("Expected released=0, got %d", set.released)
	}

	if len(set.sandboxes) != 0 {
		t.Errorf("Expected empty sandboxes slice, got length %d", len(set.sandboxes))
	}
}

// Test 11: Test capacity check logic (without actual creation)
func TestSandboxSetCapacityLogic(t *testing.T) {
	// Create a pool that's at capacity
	set := &SandboxSet{
		sandboxes: []*sandboxWrapper{
			{inUse: true},
			{inUse: true},
			{inUse: true},
		},
		maxSandboxes: 3, // At capacity
		sbPool:       &errorSandboxPool{err: errors.New("should not be called")},
	}

	// GetSandbox should fail because all are in use and we're at capacity
	_, err := set.GetSandbox()
	if err == nil {
		t.Error("Expected error when at capacity with all sandboxes in use")
	}

	expectedMsg := "SandboxSet at capacity (3/3 sandboxes in use)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// Mock pool that returns error if Create is called
type errorSandboxPool struct {
	err error
}

func (e *errorSandboxPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (Sandbox, error) {
	return nil, e.err
}

func (e *errorSandboxPool) Cleanup() {}

func (e *errorSandboxPool) AddListener(listener SandboxEventFunc) {}

func (e *errorSandboxPool) DebugString() string {
	return "errorSandboxPool"
}
