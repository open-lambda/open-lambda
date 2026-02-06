package sandbox

import (
	"fmt"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
)

// Run this with: go test -v -run TestSandboxSetIntegration
// Requires OpenLambda to be initialized

func TestSandboxSetIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize config (required for OpenLambda)
	if err := common.LoadDefaults("../../../"); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create sandbox pool
	pool, err := SandboxPoolFromConfig("test-worker", 1024)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Cleanup()

	// Create SandboxSet
	meta := &SandboxMeta{
		Runtime:    common.RT_PYTHON,
		MemLimitMB: 128,
		CPUPercent: 100,
		Installs:   []string{},
		Imports:    []string{},
	}

	config := &SandboxSetConfig{
		MaxSandboxes:         3,
		HealthCheckOnRelease: true,
		HealthCheckTimeout:   2 * time.Second,
	}

	set := NewSandboxSetWithConfig(
		pool,
		meta,
		true,
		"/tmp/ol-test-code",
		"/tmp/ol-test-scratch",
		config,
	)
	defer set.Destroy()

	t.Run("Successful execution and release", func(t *testing.T) {
		testSuccessfulExecution(t, set)
	})

	t.Run("Pool growth under load", func(t *testing.T) {
		testPoolGrowth(t, set)
	})

	t.Run("Broken sandbox removal", func(t *testing.T) {
		testBrokenSandbox(t, set)
	})

	t.Run("Capacity limit", func(t *testing.T) {
		testCapacityLimit(t, set)
	})

	// Print final stats
	t.Logf("Final stats: %v", set.Stats())
}

func testSuccessfulExecution(t *testing.T, set *SandboxSet) {
	// Get sandbox
	sb, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("GetSandbox failed: %v", err)
	}

	t.Logf("Got sandbox: %s", sb.ID())

	// Verify it's marked in-use
	stats := set.Stats()
	if stats["in_use"] != 1 {
		t.Errorf("Expected 1 in use, got %d", stats["in_use"])
	}

	// Simulate successful execution
	time.Sleep(100 * time.Millisecond)

	// Release
	err = set.ReleaseSandbox(sb)
	if err != nil {
		t.Fatalf("ReleaseSandbox failed: %v", err)
	}

	t.Logf("Released sandbox: %s", sb.ID())

	// Verify it's available
	stats = set.Stats()
	if stats["idle"] != 1 {
		t.Errorf("Expected 1 idle, got %d", stats["idle"])
	}

	// Get again - should reuse
	sb2, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("Second GetSandbox failed: %v", err)
	}

	if sb.ID() != sb2.ID() {
		t.Errorf("Expected reuse, got different sandbox")
	}

	t.Logf("Reused sandbox: %s", sb2.ID())

	set.ReleaseSandbox(sb2)
}

func testPoolGrowth(t *testing.T, set *SandboxSet) {
	initial := set.Size()
	t.Logf("Initial pool size: %d", initial)

	// Get multiple sandboxes concurrently
	sandboxes := make([]Sandbox, 3)
	for i := 0; i < 3; i++ {
		sb, err := set.GetSandbox()
		if err != nil {
			t.Fatalf("GetSandbox %d failed: %v", i, err)
		}
		sandboxes[i] = sb
		t.Logf("Got sandbox %d: %s", i, sb.ID())
	}

	// Pool should have grown
	if set.Size() < 3 {
		t.Errorf("Expected pool size >= 3, got %d", set.Size())
	}

	stats := set.Stats()
	t.Logf("Pool grown - total: %d, in_use: %d", stats["total"], stats["in_use"])

	// Release all
	for i, sb := range sandboxes {
		if err := set.ReleaseSandbox(sb); err != nil {
			t.Errorf("Release %d failed: %v", i, err)
		}
	}

	// Pool should stay same size (reuse)
	finalSize := set.Size()
	if finalSize != 3 {
		t.Errorf("Expected pool size 3 after release, got %d", finalSize)
	}

	t.Logf("After release - idle: %d", set.Stats()["idle"])
}

func testBrokenSandbox(t *testing.T, set *SandboxSet) {
	// Get a sandbox
	sb, err := set.GetSandbox()
	if err != nil {
		t.Fatalf("GetSandbox failed: %v", err)
	}

	initialSize := set.Size()
	t.Logf("Got sandbox to break: %s (pool size: %d)", sb.ID(), initialSize)

	// Simulate breaking it - destroy manually
	sb.Destroy("simulated crash")
	t.Logf("Destroyed sandbox: %s", sb.ID())

	// Remove from pool
	err = set.DestroyAndRemove(sb)
	if err != nil {
		t.Fatalf("DestroyAndRemove failed: %v", err)
	}

	// Pool should shrink
	if set.Size() >= initialSize {
		t.Errorf("Expected pool to shrink from %d, got %d", initialSize, set.Size())
	}

	stats := set.Stats()
	t.Logf("After removal - total: %d, removed: %d", stats["total"], stats["removed"])

	if stats["removed"] < 1 {
		t.Errorf("Expected removed >= 1, got %d", stats["removed"])
	}
}

func testCapacityLimit(t *testing.T, set *SandboxSet) {
	// Try to exceed capacity
	max := set.config.MaxSandboxes
	sandboxes := make([]Sandbox, 0, max+1)

	// Get up to max
	for i := 0; i < max; i++ {
		sb, err := set.GetSandbox()
		if err != nil {
			t.Fatalf("GetSandbox %d failed: %v", i, err)
		}
		sandboxes = append(sandboxes, sb)
	}

	t.Logf("Got %d sandboxes (max: %d)", len(sandboxes), max)

	// Try to get one more - should fail
	sb, err := set.GetSandbox()
	if err == nil {
		t.Errorf("Expected error at capacity, got sandbox: %s", sb.ID())
		set.ReleaseSandbox(sb)
	} else {
		t.Logf("Correctly rejected at capacity: %v", err)
	}

	// Release all
	for _, sb := range sandboxes {
		set.ReleaseSandbox(sb)
	}
}
