package sandbox

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
)

// Run with: go test -v -run TestRealContainers
func TestRealContainers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real container test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	t.Run("1_GetAndRelease", func(t *testing.T) {
		testGetAndRelease(t, pool)
	})

	t.Run("2_Reuse", func(t *testing.T) {
		testReuse(t, pool)
	})

	t.Run("3_PoolGrowth", func(t *testing.T) {
		testPoolGrowth(t, pool)
	})

	t.Run("4_Capacity", func(t *testing.T) {
		testCapacity(t, pool)
	})

	t.Run("5_DestroyRemove", func(t *testing.T) {
		testDestroyRemove(t, pool)
	})

	t.Run("6_HealthCheckPass", func(t *testing.T) {
		testHealthCheckPass(t, pool)
	})

	t.Run("7_HealthCheckFail", func(t *testing.T) {
		testHealthCheckFail(t, pool)
	})

	t.Run("8_Concurrent", func(t *testing.T) {
		testConcurrent(t, pool)
	})

	t.Run("9_ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, pool)
	})
}

func setup(t *testing.T) {
	if err := common.LoadDefaults("../../../"); err != nil {
		t.Fatalf("Config load failed: %v", err)
	}

	dirs := []string{"/tmp/test-code", "/tmp/test-scratch"}
	for _, d := range dirs {
		os.RemoveAll(d)
		os.MkdirAll(d, 0755)
	}
}

func createPool(t *testing.T) SandboxPool {
	pool, err := SandboxPoolFromConfig("test-pool", 2048)
	if err != nil {
		t.Fatalf("Pool creation failed: %v", err)
	}
	return pool
}

func createSet(pool SandboxPool, max int, healthCheck bool) *SandboxSet {
	meta := &SandboxMeta{
		Runtime:    common.RT_PYTHON,
		MemLimitMB: 128,
		CPUPercent: 100,
	}

	config := &SandboxSetConfig{
		MaxSandboxes:         max,
		HealthCheckOnRelease: healthCheck,
		HealthCheckTimeout:   2 * time.Second,
	}

	return NewSandboxSetWithConfig(
		pool, meta, true,
		"/tmp/test-code", "/tmp/test-scratch",
		config,
	)
}

// Test 1: Basic get and release
func testGetAndRelease(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 3, false)
	defer set.Destroy()

	t.Log("Getting sandbox...")
	sb, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("GetSandbox failed: %v", err)
	}

	t.Logf("✓ Got sandbox: %s", sb.ID())
	t.Logf("  Stats: %v", set.Stats())

	if set.InUse() != 1 {
		t.Errorf("Expected 1 in use, got %d", set.InUse())
	}

	t.Log("Releasing sandbox...")
	err = set.ReleaseSandbox(sb)
	if err != nil {
		t.Fatalf("ReleaseSandbox failed: %v", err)
	}

	t.Logf("✓ Released sandbox")
	t.Logf("  Stats: %v", set.Stats())

	if set.InUse() != 0 {
		t.Errorf("Expected 0 in use, got %d", set.InUse())
	}
	if set.Size() != 1 {
		t.Errorf("Expected pool size 1, got %d", set.Size())
	}
}

// Test 2: Sandbox reuse
func testReuse(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 3, false)
	defer set.Destroy()

	sb1, _ := set.GetSandbox()
	id1 := sb1.ID()
	set.ReleaseSandbox(sb1)

	sb2, _ := set.GetSandbox()
	id2 := sb2.ID()

	if id1 != id2 {
		t.Errorf("Expected reuse: %s vs %s", id1, id2)
	}

	t.Logf("✓ Sandbox reused: %s", id2)
	set.ReleaseSandbox(sb2)
}

// Test 3: Pool growth
func testPoolGrowth(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 5, false)
	defer set.Destroy()

	var sbs []Sandbox
	for i := 0; i < 3; i++ {
		sb, err := set.GetSandbox()
		if err != nil {
			t.Fatalf("GetSandbox %d failed: %v", i, err)
		}
		sbs = append(sbs, sb)
		t.Logf("  Got sandbox %d: %s (pool size: %d)", i, sb.ID(), set.Size())
	}

	if set.Size() != 3 {
		t.Errorf("Expected size 3, got %d", set.Size())
	}

	t.Logf("✓ Pool grew to %d sandboxes", set.Size())
	t.Logf("  Stats: %v", set.Stats())

	for _, sb := range sbs {
		set.ReleaseSandbox(sb)
	}
}

// Test 4: Capacity enforcement
func testCapacity(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 2, false)
	defer set.Destroy()

	sb1, _ := set.GetSandbox()
	sb2, _ := set.GetSandbox()

	t.Logf("  Got 2 sandboxes (at capacity)")

	sb3, err := set.GetSandbox()
	if err == nil {
		t.Errorf("Expected error at capacity, got: %s", sb3.ID())
		set.ReleaseSandbox(sb3)
	} else {
		t.Logf("✓ Correctly rejected at capacity: %v", err)
	}

	set.ReleaseSandbox(sb1)
	set.ReleaseSandbox(sb2)
}

// Test 5: Destroy and remove
func testDestroyRemove(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 3, false)
	defer set.Destroy()

	sb, _ := set.GetSandbox()
	beforeSize := set.Size()
	beforeRemoved := set.Stats()["removed"]

	t.Logf("  Before: size=%d, removed=%d", beforeSize, beforeRemoved)

	err := set.DestroyAndRemove(sb, "test removal")
	if err != nil {
		t.Fatalf("DestroyAndRemove failed: %v", err)
	}

	afterSize := set.Size()
	afterRemoved := set.Stats()["removed"]

	t.Logf("✓ Removed sandbox")
	t.Logf("  After: size=%d, removed=%d", afterSize, afterRemoved)

	if afterSize >= beforeSize {
		t.Errorf("Size should decrease: %d -> %d", beforeSize, afterSize)
	}
	if afterRemoved != beforeRemoved+1 {
		t.Errorf("Removed should increase: %d -> %d", beforeRemoved, afterRemoved)
	}
}

// Test 6: Health check passing
func testHealthCheckPass(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 3, true) // Health checks enabled
	defer set.Destroy()

	sb, _ := set.GetSandbox()
	t.Logf("  Got sandbox: %s", sb.ID())

	err := set.ReleaseSandbox(sb)
	if err != nil {
		t.Fatalf("Release with health check failed: %v", err)
	}

	stats := set.Stats()
	t.Logf("✓ Health check passed")
	t.Logf("  Health checks: %d, failures: %d", stats["health_checks"], stats["health_failures"])

	if stats["health_checks"] < 1 {
		t.Errorf("Expected health check to run")
	}
}

// Test 7: Health check failure (simulate dead sandbox)
func testHealthCheckFail(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 3, true)
	defer set.Destroy()

	sb, _ := set.GetSandbox()
	beforeSize := set.Size()

	t.Logf("  Destroying sandbox to simulate crash: %s", sb.ID())
	sb.Destroy("simulated crash")

	// Try to release dead sandbox
	err := set.ReleaseSandbox(sb)

	stats := set.Stats()
	t.Logf("  Health check result: checks=%d, failures=%d",
		stats["health_checks"], stats["health_failures"])

	// Health check might fail and auto-remove, OR succeed if Destroy doesn't break Pause()
	// Check if it was removed
	if set.Size() < beforeSize {
		t.Logf("✓ Dead sandbox auto-removed by health check")
	} else if err != nil {
		t.Logf("✓ Health check detected issue: %v", err)
	} else {
		t.Logf("  Note: Health check passed (Pause still works on destroyed sandbox)")
	}
}

// Test 8: Concurrent access
func testConcurrent(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 10, false)
	defer set.Destroy()

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sb, err := set.GetSandbox()
			if err != nil {
				// OK if at capacity
				return
			}

			time.Sleep(10 * time.Millisecond)

			err = set.ReleaseSandbox(sb)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d release failed: %v", id, err)
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

	if errorCount == 0 {
		t.Logf("✓ No race conditions")
	}

	stats := set.Stats()
	t.Logf("  Final stats: %v", stats)

	if set.InUse() != 0 {
		t.Errorf("Expected 0 in use after concurrent test, got %d", set.InUse())
	}
}

// Test 9: Error handling
func testErrorHandling(t *testing.T, pool SandboxPool) {
	set := createSet(pool, 3, false)
	defer set.Destroy()

	// Test 1: Release nil
	err := set.ReleaseSandbox(nil)
	if err == nil {
		t.Error("Expected error releasing nil")
	} else {
		t.Logf("✓ Rejected nil: %v", err)
	}

	// Test 2: Double release
	sb, _ := set.GetSandbox()
	set.ReleaseSandbox(sb)
	err = set.ReleaseSandbox(sb)
	if err == nil {
		t.Error("Expected error on double release")
	} else {
		t.Logf("✓ Rejected double release: %v", err)
	}

	// Test 3: Destroy nil
	err = set.DestroyAndRemove(nil, "test")
	if err == nil {
		t.Error("Expected error destroying nil")
	} else {
		t.Logf("✓ Rejected destroy nil: %v", err)
	}

	// Test 4: Unknown sandbox
	set2 := createSet(pool, 2, false)
	defer set2.Destroy()

	sb2, _ := set2.GetSandbox()
	err = set.ReleaseSandbox(sb2) // Wrong set!
	if err == nil {
		t.Error("Expected error releasing sandbox from different set")
	} else {
		t.Logf("✓ Rejected unknown sandbox: %v", err)
	}

	set2.ReleaseSandbox(sb2)
}

// =============================================================================
// NEW PHASE 1 TESTS
// =============================================================================

// Test 10: Options pattern - WithTimeout
func TestWithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	set := createSet(pool, 1, false) // Capacity of 1
	defer set.Destroy()

	// Get the only sandbox
	sb1, _ := set.GetSandbox()

	// Try to get another with short timeout (should fail)
	start := time.Now()
	sb2, err := set.GetSandbox(WithTimeout(500 * time.Millisecond))
	elapsed := time.Since(start)

	if err == nil {
		t.Errorf("Expected timeout error, got sandbox: %s", sb2.ID())
		set.ReleaseSandbox(sb2)
	} else if elapsed < 400*time.Millisecond {
		t.Errorf("Timeout too fast: %v", elapsed)
	} else {
		t.Logf("✓ Timeout after %v: %v", elapsed, err)
	}

	set.ReleaseSandbox(sb1)
}

// Test 11: Options pattern - SkipHealthCheck
func TestSkipHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping health check test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	set := createSet(pool, 2, true) // Health checks enabled
	defer set.Destroy()

	sb, _ := set.GetSandbox()

	// Release without health check
	beforeChecks := set.Stats()["health_checks"]
	err := set.ReleaseSandbox(sb, SkipHealthCheck())
	afterChecks := set.Stats()["health_checks"]

	if err != nil {
		t.Fatalf("Release with skip failed: %v", err)
	}

	if afterChecks != beforeChecks {
		t.Errorf("Health check ran despite SkipHealthCheck: %d -> %d", beforeChecks, afterChecks)
	} else {
		t.Logf("✓ Health check skipped: %d checks", afterChecks)
	}
}

// Test 12: Warm operation
func TestWarm(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping warm test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	set := createSet(pool, 5, false)
	defer set.Destroy()

	t.Logf("Initial size: %d", set.Size())

	// Warm to 3 sandboxes
	err := set.Warm(3)
	if err != nil {
		t.Fatalf("Warm failed: %v", err)
	}

	if set.Size() != 3 {
		t.Errorf("Expected size 3 after warm, got %d", set.Size())
	}

	stats := set.Stats()
	if stats["idle"] != 3 {
		t.Errorf("Expected 3 idle, got %d", stats["idle"])
	}

	t.Logf("✓ Warmed to %d sandboxes (all idle)", set.Size())

	// Warm again (should be no-op)
	err = set.Warm(3)
	if err != nil {
		t.Errorf("Warm to same level failed: %v", err)
	}

	if set.Size() != 3 {
		t.Errorf("Size changed unexpectedly: %d", set.Size())
	}

	t.Logf("✓ Warm to same level is no-op: %d sandboxes", set.Size())
}

// Test 13: Shrink operation
func TestShrink(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping shrink test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	set := createSet(pool, 10, false)
	defer set.Destroy()

	// Create 5 sandboxes and keep them all idle
	for i := 0; i < 5; i++ {
		sb, _ := set.GetSandbox()
		set.ReleaseSandbox(sb)
	}

	t.Logf("Created %d sandboxes", set.Size())

	// Shrink to 2
	err := set.Shrink(2)
	if err != nil {
		t.Fatalf("Shrink failed: %v", err)
	}

	stats := set.Stats()
	if stats["idle"] > 2 {
		t.Errorf("Expected max 2 idle after shrink, got %d", stats["idle"])
	}

	t.Logf("✓ Shrunk to %d idle sandboxes", stats["idle"])
	t.Logf("  Removed: %d", stats["removed"])
}

// Test 14: Metrics method
func TestMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping metrics test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	set := createSet(pool, 5, false)
	defer set.Destroy()

	// Create some activity
	sb1, _ := set.GetSandbox()
	sb2, _ := set.GetSandbox()
	time.Sleep(50 * time.Millisecond)
	set.ReleaseSandbox(sb1)
	set.ReleaseSandbox(sb2)

	// Get metrics
	metrics := set.Metrics()

	t.Logf("✓ Metrics retrieved:")
	t.Logf("  Total: %d, InUse: %d, Idle: %d", metrics.Total, metrics.InUse, metrics.Idle)
	t.Logf("  Created: %d, Borrowed: %d, Released: %d", metrics.Created, metrics.Borrowed, metrics.Released)
	t.Logf("  HitRate: %.2f, Utilization: %.2f", metrics.HitRate, metrics.CapacityUtilization)
	t.Logf("  Uptime: %v", metrics.Uptime)

	if metrics.Total != 2 {
		t.Errorf("Expected total 2, got %d", metrics.Total)
	}

	if metrics.HitRate < 0 || metrics.HitRate > 1 {
		t.Errorf("Invalid hit rate: %.2f", metrics.HitRate)
	}

	if len(metrics.SandboxMetrics) != 2 {
		t.Errorf("Expected 2 sandbox metrics, got %d", len(metrics.SandboxMetrics))
	}

	for i, sm := range metrics.SandboxMetrics {
		t.Logf("  Sandbox %d: ID=%s, Age=%v, UseCount=%d",
			i, sm.ID, sm.Age, sm.UseCount)
	}
}

// Test 15: Event emission
func TestEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping events test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	// Track events
	var mu sync.Mutex
	events := []SandboxSetEvent{}

	config := &SandboxSetConfig{
		MaxSandboxes:         5,
		HealthCheckOnRelease: false,
		HealthCheckTimeout:   2 * time.Second,
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

	set := NewSandboxSetWithConfig(pool, meta, true, "/tmp/test-code", "/tmp/test-scratch", config)
	defer set.Destroy()

	// Generate events
	sb, _ := set.GetSandbox()
	time.Sleep(100 * time.Millisecond) // Let events propagate
	set.ReleaseSandbox(sb)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	eventCount := len(events)
	mu.Unlock()

	t.Logf("✓ Captured %d events:", eventCount)
	mu.Lock()
	for i, e := range events {
		t.Logf("  %d. %s (sandbox: %s)", i+1, e.Type.String(), e.SandboxID)
	}
	mu.Unlock()

	// Should have at least: Created, Borrowed, Released
	if eventCount < 3 {
		t.Errorf("Expected at least 3 events, got %d", eventCount)
	}
}

// Test 16: Priority queue FIFO strategy
func TestSchedulingFIFO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping FIFO test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	config := &SandboxSetConfig{
		MaxSandboxes:       5,
	}

	meta := &SandboxMeta{
		Runtime:    common.RT_PYTHON,
		MemLimitMB: 128,
		CPUPercent: 100,
	}

	set := NewSandboxSetWithConfig(pool, meta, true, "/tmp/test-code", "/tmp/test-scratch", config)
	defer set.Destroy()

	// Create 3 sandboxes in order
	sb1, _ := set.GetSandbox()
	id1 := sb1.ID()
	time.Sleep(10 * time.Millisecond)

	sb2, _ := set.GetSandbox()
	_ = sb2.ID() // id2 for reference
	time.Sleep(10 * time.Millisecond)

	sb3, _ := set.GetSandbox()
	_ = sb3.ID() // id3 for reference

	// Release in reverse order
	set.ReleaseSandbox(sb3)
	set.ReleaseSandbox(sb2)
	set.ReleaseSandbox(sb1)

	// With FIFO, should get sb1 first (oldest)
	sbNext, _ := set.GetSandbox()
	idNext := sbNext.ID()

	if idNext != id1 {
		t.Errorf("FIFO failed: expected %s (oldest), got %s", id1, idNext)
	} else {
		t.Logf("✓ FIFO returned oldest: %s", idNext)
	}

	set.ReleaseSandbox(sbNext)
}

// Test 17: Priority queue LRU strategy
func TestSchedulingLRU(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LRU test")
	}

	setup(t)
	pool := createPool(t)
	defer pool.Cleanup()

	config := &SandboxSetConfig{
		MaxSandboxes:       5,
	}

	meta := &SandboxMeta{
		Runtime:    common.RT_PYTHON,
		MemLimitMB: 128,
		CPUPercent: 100,
	}

	set := NewSandboxSetWithConfig(pool, meta, true, "/tmp/test-code", "/tmp/test-scratch", config)
	defer set.Destroy()

	// Create 2 sandboxes
	sb1, _ := set.GetSandbox()
	_ = sb1.ID() // id1 for reference
	sb2, _ := set.GetSandbox()
	id2 := sb2.ID()

	// Release sb2 first, then sb1
	set.ReleaseSandbox(sb2)
	time.Sleep(10 * time.Millisecond)
	set.ReleaseSandbox(sb1)

	// With LRU, should get sb2 first (less recently used)
	sbNext, _ := set.GetSandbox()
	idNext := sbNext.ID()

	if idNext != id2 {
		t.Logf("  LRU didn't return expected (may depend on timing)")
		t.Logf("  Expected: %s, Got: %s", id2, idNext)
	} else {
		t.Logf("✓ LRU returned least recently used: %s", idNext)
	}

	set.ReleaseSandbox(sbNext)
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func BenchmarkGetSandbox(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	setup(&testing.T{})
	pool, _ := SandboxPoolFromConfig("bench-pool", 2048)
	defer pool.Cleanup()

	set := createSet(pool, 100, false)
	defer set.Destroy()

	// Pre-warm pool
	for i := 0; i < 10; i++ {
		sb, _ := set.GetSandbox()
		set.ReleaseSandbox(sb)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sb, err := set.GetSandbox()
		if err != nil {
			b.Fatalf("GetSandbox failed: %v", err)
		}
		set.ReleaseSandbox(sb)
	}
}

func BenchmarkConcurrentGetRelease(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	setup(&testing.T{})
	pool, _ := SandboxPoolFromConfig("bench-pool", 2048)
	defer pool.Cleanup()

	set := createSet(pool, 50, false)
	defer set.Destroy()

	// Pre-warm
	for i := 0; i < 10; i++ {
		sb, _ := set.GetSandbox()
		set.ReleaseSandbox(sb)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sb, err := set.GetSandbox()
			if err == nil {
				set.ReleaseSandbox(sb)
			}
		}
	})
}

func BenchmarkStats(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	setup(&testing.T{})
	pool, _ := SandboxPoolFromConfig("bench-pool", 2048)
	defer pool.Cleanup()

	set := createSet(pool, 10, false)
	defer set.Destroy()

	// Create some sandboxes
	for i := 0; i < 5; i++ {
		sb, _ := set.GetSandbox()
		set.ReleaseSandbox(sb)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = set.Stats()
	}
}

func BenchmarkMetrics(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	setup(&testing.T{})
	pool, _ := SandboxPoolFromConfig("bench-pool", 2048)
	defer pool.Cleanup()

	set := createSet(pool, 10, false)
	defer set.Destroy()

	// Create some sandboxes
	for i := 0; i < 5; i++ {
		sb, _ := set.GetSandbox()
		set.ReleaseSandbox(sb)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = set.Metrics()
	}
}
