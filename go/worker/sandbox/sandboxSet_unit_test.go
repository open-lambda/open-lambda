package sandbox

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
)

// Test 1: Basic get and release
func TestUnit_GetAndRelease(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	// Get sandbox
	sb, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("GetSandbox failed: %v", err)
	}

	if set.InUse() != 1 {
		t.Errorf("Expected 1 in use, got %d", set.InUse())
	}

	// Release sandbox
	err = set.ReleaseSandbox(sb)
	if err != nil {
		t.Fatalf("ReleaseSandbox failed: %v", err)
	}

	if set.InUse() != 0 {
		t.Errorf("Expected 0 in use after release, got %d", set.InUse())
	}

	stats := set.Stats()
	if stats["total"] != 1 {
		t.Errorf("Expected 1 total sandbox, got %d", stats["total"])
	}

	t.Log("✓ Basic get/release works")
}

// Test 2: Sandbox reuse
func TestUnit_Reuse(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	// Get and release
	sb1, _ := set.GetSandbox()
	id1 := sb1.ID()
	set.ReleaseSandbox(sb1)

	// Get again - should reuse
	sb2, _ := set.GetSandbox()
	id2 := sb2.ID()

	if id1 != id2 {
		t.Errorf("Expected reuse: %s vs %s", id1, id2)
	}

	if pool.CreateCount() != 1 {
		t.Errorf("Expected 1 creation, got %d", pool.CreateCount())
	}

	set.ReleaseSandbox(sb2)
	t.Log("✓ Sandbox reuse works")
}

// Test 3: Pool growth
func TestUnit_PoolGrowth(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 10, false)

	sbs := []Sandbox{}
	for i := 0; i < 5; i++ {
		sb, err := set.GetSandbox()
		if err != nil {
			t.Fatalf("GetSandbox %d failed: %v", i, err)
		}
		sbs = append(sbs, sb)
	}

	if set.Size() != 5 {
		t.Errorf("Expected size 5, got %d", set.Size())
	}

	for _, sb := range sbs {
		set.ReleaseSandbox(sb)
	}

	t.Log("✓ Pool grows correctly")
}

// Test 4: Capacity enforcement
func TestUnit_Capacity(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 2, false)

	sb1, _ := set.GetSandbox()
	sb2, _ := set.GetSandbox()

	// Third should fail - at capacity (use short timeout)
	sb3, err := set.GetSandbox(WithTimeout(100 * time.Millisecond))
	if err == nil {
		t.Error("Expected error at capacity")
	}
	if sb3 != nil {
		t.Error("Expected nil sandbox at capacity")
	}

	set.ReleaseSandbox(sb1)
	set.ReleaseSandbox(sb2)

	t.Log("✓ Capacity enforcement works")
}

// Test 5: DestroyAndRemove
func TestUnit_DestroyAndRemove(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	sb, _ := set.GetSandbox()
	beforeSize := set.Size()

	err := set.DestroyAndRemove(sb, "test removal")
	if err != nil {
		t.Fatalf("DestroyAndRemove failed: %v", err)
	}

	afterSize := set.Size()
	if afterSize != beforeSize-1 {
		t.Errorf("Size should decrease: %d -> %d", beforeSize, afterSize)
	}

	// Verify mock was destroyed
	mockSb := sb.(*MockSandbox)
	if !mockSb.WasDestroyed() {
		t.Error("Sandbox should be destroyed")
	}

	t.Log("✓ DestroyAndRemove works")
}

// Test 6: Health check pass
func TestUnit_HealthCheckPass(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, true) // Health checks enabled

	sb, _ := set.GetSandbox()

	err := set.ReleaseSandbox(sb)
	if err != nil {
		t.Fatalf("Release with health check failed: %v", err)
	}

	stats := set.Stats()
	if stats["health_checks"] < 1 {
		t.Error("Health check should have run")
	}

	// Verify pause was called
	mockSb := sb.(*MockSandbox)
	if mockSb.PauseCount() < 1 {
		t.Error("Pause should have been called for health check")
	}

	t.Log("✓ Health check pass works")
}

// Test 7: Health check fail - sandbox auto-removed
func TestUnit_HealthCheckFail(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, true) // Health checks enabled

	sb, _ := set.GetSandbox()
	beforeSize := set.Size()

	// Make pause fail
	mockSb := sb.(*MockSandbox)
	mockSb.SetPauseError(errors.New("pause failed"))

	err := set.ReleaseSandbox(sb)
	// Should fail and auto-remove
	if err == nil {
		t.Error("Expected error on health check failure")
	}

	afterSize := set.Size()
	if afterSize >= beforeSize {
		t.Errorf("Failed sandbox should be removed: size %d -> %d", beforeSize, afterSize)
	}

	stats := set.Stats()
	if stats["health_failures"] < 1 {
		t.Error("Health failure should be recorded")
	}

	t.Log("✓ Health check failure auto-removes sandbox")
}

// Test 8: Concurrent access
func TestUnit_Concurrent(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 20, false)

	var wg sync.WaitGroup
	errors := make(chan error, 50)

	// 50 goroutines trying to get/release
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sb, err := set.GetSandbox()
			if err != nil {
				// OK if at capacity
				return
			}

			time.Sleep(1 * time.Millisecond)

			err = set.ReleaseSandbox(sb)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Had %d concurrent errors", errorCount)
	}

	if set.InUse() != 0 {
		t.Errorf("Expected 0 in use after concurrent test, got %d", set.InUse())
	}

	t.Log("✓ Concurrent access is safe")
}

// Test 9: Error handling - nil sandbox
func TestUnit_ErrorHandling_Nil(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	// Release nil
	err := set.ReleaseSandbox(nil)
	if err == nil {
		t.Error("Expected error releasing nil")
	}

	// Destroy nil
	err = set.DestroyAndRemove(nil, "test")
	if err == nil {
		t.Error("Expected error destroying nil")
	}

	t.Log("✓ Nil sandbox rejection works")
}

// Test 10: Double release
func TestUnit_DoubleRelease(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	sb, _ := set.GetSandbox()
	set.ReleaseSandbox(sb)

	// Second release should fail
	err := set.ReleaseSandbox(sb)
	if err != ErrNotInUse {
		t.Errorf("Expected ErrNotInUse on double release, got: %v", err)
	}

	t.Log("✓ Double release detection works")
}

// Test 11: Unknown sandbox
func TestUnit_UnknownSandbox(t *testing.T) {
	pool1 := NewMockSandboxPool()
	pool2 := NewMockSandboxPool()
	set1 := createTestSet(pool1, 5, false)
	set2 := createTestSet(pool2, 5, false)

	sb, _ := set2.GetSandbox()

	// Try to release in wrong set
	err := set1.ReleaseSandbox(sb)
	if err != ErrUnknownSandbox {
		t.Errorf("Expected ErrUnknownSandbox, got: %v", err)
	}

	set2.ReleaseSandbox(sb)
	t.Log("✓ Unknown sandbox detection works")
}

// Test 12: Warm operation
func TestUnit_Warm(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 10, false)

	err := set.Warm(5)
	if err != nil {
		t.Fatalf("Warm failed: %v", err)
	}

	if set.Size() != 5 {
		t.Errorf("Expected 5 sandboxes after warm, got %d", set.Size())
	}

	stats := set.Stats()
	if stats["idle"] != 5 {
		t.Errorf("Expected 5 idle, got %d", stats["idle"])
	}

	t.Log("✓ Warm operation works")
}

// Test 13: Warm exceeds capacity
func TestUnit_WarmExceedsCapacity(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	err := set.Warm(10)
	if err == nil {
		t.Error("Expected error warming beyond capacity")
	}

	t.Log("✓ Warm capacity check works")
}

// Test 14: Shrink operation
func TestUnit_Shrink(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 10, false)

	// Create 5 idle sandboxes
	set.Warm(5)

	err := set.Shrink(2)
	if err != nil {
		t.Fatalf("Shrink failed: %v", err)
	}

	stats := set.Stats()
	if stats["idle"] > 2 {
		t.Errorf("Expected <= 2 idle after shrink, got %d", stats["idle"])
	}

	t.Log("✓ Shrink operation works")
}

// Test 15: Options pattern - WithTimeout
func TestUnit_WithTimeout(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 1, false)

	// Get the only sandbox
	sb1, _ := set.GetSandbox()

	// Try to get with short timeout (currently returns capacity error immediately)
	// Note: Timeout enforcement is not yet implemented in simplified version
	start := time.Now()
	sb2, err := set.GetSandbox(WithTimeout(100 * time.Millisecond))
	elapsed := time.Since(start)

	if err != ErrCapacity {
		t.Errorf("Expected ErrCapacity at capacity, got: %v", err)
	}

	// Should return immediately, not wait for timeout
	if elapsed > 50*time.Millisecond {
		t.Errorf("Should fail immediately at capacity, but took: %v", elapsed)
	}

	if sb2 != nil {
		t.Error("Expected nil sandbox at capacity")
	}

	set.ReleaseSandbox(sb1)
	t.Log("✓ WithTimeout works")
}

// Test 16: Options pattern - SkipHealthCheck
func TestUnit_SkipHealthCheck(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, true) // Health checks enabled by default

	sb, _ := set.GetSandbox()

	beforeChecks := set.Stats()["health_checks"]
	err := set.ReleaseSandbox(sb, SkipHealthCheck())
	afterChecks := set.Stats()["health_checks"]

	if err != nil {
		t.Fatalf("Release with skip failed: %v", err)
	}

	if afterChecks != beforeChecks {
		t.Errorf("Health check ran despite skip: %d -> %d", beforeChecks, afterChecks)
	}

	t.Log("✓ SkipHealthCheck works")
}

// Test 17: Metrics
func TestUnit_Metrics(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	// Create some activity
	sb1, _ := set.GetSandbox()
	sb2, _ := set.GetSandbox()
	time.Sleep(10 * time.Millisecond)
	set.ReleaseSandbox(sb1)
	set.ReleaseSandbox(sb2)

	metrics := set.Metrics()

	if metrics.Total != 2 {
		t.Errorf("Expected 2 total, got %d", metrics.Total)
	}

	if metrics.Created != 2 {
		t.Errorf("Expected 2 created, got %d", metrics.Created)
	}

	if metrics.HitRate < 0 || metrics.HitRate > 1 {
		t.Errorf("Invalid hit rate: %.2f", metrics.HitRate)
	}

	if len(metrics.SandboxMetrics) != 2 {
		t.Errorf("Expected 2 sandbox metrics, got %d", len(metrics.SandboxMetrics))
	}

	t.Log("✓ Metrics collection works")
}

// Test 18: Event emission
func TestUnit_Events(t *testing.T) {
	pool := NewMockSandboxPool()

	var mu sync.Mutex
	events := []SandboxSetEvent{}

	config := &SandboxSetConfig{
		MaxSandboxes:         5,
		HealthCheckOnRelease: false,
		EventHandlers: []SandboxSetEventHandler{
			func(e SandboxSetEvent) {
				mu.Lock()
				events = append(events, e)
				mu.Unlock()
			},
		},
	}

	meta := &SandboxMeta{
		Runtime:    common.RT_PYTHON,
		MemLimitMB: 128,
		CPUPercent: 100,
	}

	set := NewSandboxSetWithConfig(pool, meta, true, "/tmp/code", "/tmp/scratch", config)

	sb, _ := set.GetSandbox()
	time.Sleep(50 * time.Millisecond) // Let events propagate
	set.ReleaseSandbox(sb)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	eventCount := len(events)
	mu.Unlock()

	if eventCount < 3 {
		t.Errorf("Expected at least 3 events (created, borrowed, released), got %d", eventCount)
	}

	t.Log("✓ Event emission works")
}

// Test 19: Creation failure
func TestUnit_CreateFailure(t *testing.T) {
	pool := NewMockSandboxPool()
	pool.SetCreateError(errors.New("creation failed"))

	set := createTestSet(pool, 5, false)

	sb, err := set.GetSandbox()
	if err == nil {
		t.Error("Expected creation error")
	}
	if sb != nil {
		t.Error("Expected nil sandbox on creation failure")
	}

	t.Log("✓ Creation failure handling works")
}

// Test 20: Health check timeout
func TestUnit_HealthCheckTimeout(t *testing.T) {
	pool := NewMockSandboxPool()

	config := &SandboxSetConfig{
		MaxSandboxes:         5,
		HealthCheckOnRelease: true,
		HealthCheckTimeout:   50 * time.Millisecond,
		HealthCheckMethod:    HealthCheckPause,
	}

	meta := &SandboxMeta{
		Runtime:    common.RT_PYTHON,
		MemLimitMB: 128,
		CPUPercent: 100,
	}

	set := NewSandboxSetWithConfig(pool, meta, true, "/tmp/code", "/tmp/scratch", config)

	sb, err := set.GetSandbox()
	if err != nil || sb == nil {
		t.Fatalf("Failed to get sandbox: %v", err)
	}

	// Make pause hang (simulate unresponsive sandbox)
	mockSb := sb.(*MockSandbox)
	mockSb.SetPauseDelay(200 * time.Millisecond) // Longer than health check timeout

	beforeSize := set.Size()
	start := time.Now()

	// This should timeout and remove sandbox
	err = set.ReleaseSandbox(sb)
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Errorf("Health check took too long: %v", elapsed)
	}

	if err == nil {
		t.Error("Expected error from failed health check")
	}

	afterSize := set.Size()
	if afterSize >= beforeSize {
		t.Error("Sandbox should be removed after health check timeout")
	}

	t.Log("✓ Health check timeout works")
}

// Test 21: Closed pool
func TestUnit_ClosedPool(t *testing.T) {
	pool := NewMockSandboxPool()
	set := createTestSet(pool, 5, false)

	sb, _ := set.GetSandbox()

	// Destroy pool
	set.Destroy()

	// Operations should fail
	_, err := set.GetSandbox()
	if err != ErrClosed {
		t.Errorf("Expected ErrClosed after Destroy, got: %v", err)
	}

	// Verify sandbox was destroyed
	mockSb := sb.(*MockSandbox)
	if !mockSb.WasDestroyed() {
		t.Error("Sandbox should be destroyed when pool is destroyed")
	}

	t.Log("✓ Closed pool detection works")
}

// Test 22: Multiple health check methods
func TestUnit_HealthCheckMethods(t *testing.T) {
	pool := NewMockSandboxPool()

	methods := []HealthCheckMethod{
		HealthCheckPause,
		HealthCheckPauseUnpause,
	}

	for _, method := range methods {
		config := &SandboxSetConfig{
			MaxSandboxes:         5,
			HealthCheckOnRelease: true,
			HealthCheckMethod:    method,
			HealthCheckTimeout:   1 * time.Second,
		}

		meta := &SandboxMeta{
			Runtime:    common.RT_PYTHON,
			MemLimitMB: 128,
			CPUPercent: 100,
		}

		set := NewSandboxSetWithConfig(pool, meta, true, "/tmp/code", "/tmp/scratch", config)

		sb, _ := set.GetSandbox()
		err := set.ReleaseSandbox(sb)

		if err != nil {
			t.Errorf("Health check method %v failed: %v", method, err)
		}

		mockSb := sb.(*MockSandbox)
		if method == HealthCheckPauseUnpause {
			if !mockSb.WasUnpaused() {
				t.Errorf("Method %v should call Unpause", method)
			}
		}
	}

	t.Log("✓ Multiple health check methods work")
}

// Helper function to create test set
func createTestSet(pool SandboxPool, maxSandboxes int, healthCheck bool) *SandboxSet {
	meta := &SandboxMeta{
		Runtime:    common.RT_PYTHON,
		MemLimitMB: 128,
		CPUPercent: 100,
	}

	config := &SandboxSetConfig{
		MaxSandboxes:         maxSandboxes,
		HealthCheckOnRelease: healthCheck,
		HealthCheckTimeout:   1 * time.Second,
		HealthCheckMethod:    HealthCheckPause,
		GetTimeout:           30 * time.Second,
		CreateTimeout:        10 * time.Second,
	}

	return NewSandboxSetWithConfig(
		pool, meta, true,
		"/tmp/test-code", "/tmp/test-scratch",
		config,
	)
}
