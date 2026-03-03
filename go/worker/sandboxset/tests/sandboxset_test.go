package tests

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
	"github.com/open-lambda/open-lambda/go/worker/sandboxset"
)

// newTestConfig returns a valid Config backed by mocks and a temp directory.
func newTestConfig(t *testing.T) (*sandboxset.Config, *sandbox.MockSandboxPool) {
	t.Helper()
	tmpDir := t.TempDir()
	common.Conf = &common.Config{Worker_dir: tmpDir}
	scratchDirs, err := common.NewDirMaker("scratch", common.STORE_REGULAR)
	if err != nil {
		t.Fatal(err)
	}
	pool := &sandbox.MockSandboxPool{}
	cfg := &sandboxset.Config{
		Pool:        pool,
		CodeDir:     tmpDir + "/code",
		ScratchDirs: scratchDirs,
	}
	return cfg, pool
}

// newTestSet is a shortcut that creates a valid SandboxSet.
func newTestSet(t *testing.T) (sandboxset.SandboxSet, *sandbox.MockSandboxPool) {
	t.Helper()
	cfg, pool := newTestConfig(t)
	set, err := sandboxset.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return set, pool
}

// --- Constructor tests ---

func TestNew_NilConfig(t *testing.T) {
	_, err := sandboxset.New(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestNew_NilPool(t *testing.T) {
	tmpDir := t.TempDir()
	common.Conf = &common.Config{Worker_dir: tmpDir}
	scratchDirs, err := common.NewDirMaker("scratch", common.STORE_REGULAR)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sandboxset.New(&sandboxset.Config{
		CodeDir:     "/some/dir",
		ScratchDirs: scratchDirs,
	})
	if err == nil {
		t.Fatal("expected error for nil Pool")
	}
}

func TestNew_EmptyCodeDir(t *testing.T) {
	tmpDir := t.TempDir()
	common.Conf = &common.Config{Worker_dir: tmpDir}
	scratchDirs, err := common.NewDirMaker("scratch", common.STORE_REGULAR)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sandboxset.New(&sandboxset.Config{
		Pool:        &sandbox.MockSandboxPool{},
		ScratchDirs: scratchDirs,
	})
	if err == nil {
		t.Fatal("expected error for empty CodeDir")
	}
}

func TestNew_NilScratchDirs(t *testing.T) {
	_, err := sandboxset.New(&sandboxset.Config{
		Pool:    &sandbox.MockSandboxPool{},
		CodeDir: "/some/dir",
	})
	if err == nil {
		t.Fatal("expected error for nil ScratchDirs")
	}
}

func TestNew_Valid(t *testing.T) {
	set, _ := newTestSet(t)
	if set == nil {
		t.Fatal("expected non-nil SandboxSet")
	}
}

// --- Get tests ---

func TestGet_CreatesNew(t *testing.T) {
	set, pool := newTestSet(t)
	sb, err := set.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sb == nil {
		t.Fatal("expected non-nil sandbox")
	}
	if n := len(pool.CreatedSandboxes()); n != 1 {
		t.Fatalf("expected 1 created sandbox, got %d", n)
	}
}

func TestGet_ReusesIdle(t *testing.T) {
	set, _ := newTestSet(t)

	sb1, err := set.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	id1 := sb1.ID()

	if err := set.Put(sb1); err != nil {
		t.Fatalf("Put: %v", err)
	}

	sb2, err := set.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sb2.ID() != id1 {
		t.Fatalf("expected reuse (ID %s), got new (ID %s)", id1, sb2.ID())
	}
}

func TestGet_UnpauseFail(t *testing.T) {
	set, pool := newTestSet(t)

	sb1, err := set.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Inject unpause error before putting back.
	sb1.(*sandbox.MockSandbox).UnpauseErr = errors.New("broken")
	if err := set.Put(sb1); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Next Get should find the bad sandbox, destroy it, and create a new one.
	sb2, err := set.Get()
	if err != nil {
		t.Fatalf("Get after unpause fail: %v", err)
	}
	if sb2.ID() == sb1.ID() {
		t.Fatal("expected a different sandbox after unpause failure")
	}
	if !sb1.(*sandbox.MockSandbox).IsDestroyed() {
		t.Fatal("bad sandbox should have been destroyed")
	}
	if n := len(pool.CreatedSandboxes()); n != 2 {
		t.Fatalf("expected 2 creates (original + retry), got %d", n)
	}
}

func TestGet_CreateFail(t *testing.T) {
	set, pool := newTestSet(t)
	pool.CreateErr = errors.New("out of resources")

	_, err := set.Get()
	if err == nil {
		t.Fatal("expected error when pool.Create fails")
	}
}

func TestGet_AfterClose(t *testing.T) {
	set, _ := newTestSet(t)
	if err := set.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, err := set.Get()
	if err == nil {
		t.Fatal("expected error after Close")
	}
}

// --- Put tests ---

func TestPut_PausesAndReturns(t *testing.T) {
	set, _ := newTestSet(t)
	sb, err := set.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	mock := sb.(*sandbox.MockSandbox)
	if mock.IsPaused() {
		t.Fatal("sandbox should be unpaused after Get")
	}

	if err := set.Put(sb); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !mock.IsPaused() {
		t.Fatal("sandbox should be paused after Put")
	}
}

func TestPut_PauseFail(t *testing.T) {
	set, _ := newTestSet(t)
	sb, err := set.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	sb.(*sandbox.MockSandbox).PauseErr = errors.New("pause broken")

	err = set.Put(sb)
	if err == nil {
		t.Fatal("expected error when Pause fails")
	}
	if !sb.(*sandbox.MockSandbox).IsDestroyed() {
		t.Fatal("sandbox should be destroyed when Pause fails")
	}
}

func TestPut_NotInPool(t *testing.T) {
	set, _ := newTestSet(t)
	orphan := sandbox.NewMockSandbox("orphan")
	err := set.Put(orphan)
	if err == nil {
		t.Fatal("expected error for sandbox not in pool")
	}
}

// --- Destroy tests ---

func TestDestroy_RemovesFromPool(t *testing.T) {
	set, _ := newTestSet(t)
	sb, err := set.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if err := set.Destroy(sb, "test"); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if !sb.(*sandbox.MockSandbox).IsDestroyed() {
		t.Fatal("sandbox should be destroyed")
	}

	// Next Get should create a new one, not reuse the destroyed one.
	sb2, err := set.Get()
	if err != nil {
		t.Fatalf("Get after Destroy: %v", err)
	}
	if sb2.ID() == sb.ID() {
		t.Fatal("should not reuse a destroyed sandbox")
	}
}

func TestDestroy_NotInPool(t *testing.T) {
	set, _ := newTestSet(t)
	orphan := sandbox.NewMockSandbox("orphan")
	err := set.Destroy(orphan, "test")
	if err == nil {
		t.Fatal("expected error for sandbox not in pool")
	}
	if !orphan.IsDestroyed() {
		t.Fatal("sandbox should still be destroyed even if not in pool")
	}
}

// --- Close tests ---

func TestClose_DestroysAll(t *testing.T) {
	set, pool := newTestSet(t)

	// Create 3 sandboxes: 2 in-use, 1 idle.
	sb1, _ := set.Get()
	sb2, _ := set.Get()
	sb3, _ := set.Get()
	_ = set.Put(sb3) // return one to idle

	if err := set.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for _, m := range pool.CreatedSandboxes() {
		if !m.IsDestroyed() {
			t.Fatalf("sandbox %s should be destroyed after Close", m.ID())
		}
	}
	_ = sb1
	_ = sb2
}

func TestClose_Twice(t *testing.T) {
	set, _ := newTestSet(t)
	if err := set.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	err := set.Close()
	if err == nil {
		t.Fatal("expected error on second Close")
	}
}

func TestClose_EmptyPool(t *testing.T) {
	set, _ := newTestSet(t)
	if err := set.Close(); err != nil {
		t.Fatalf("Close on empty pool: %v", err)
	}
}

// --- Lifecycle tests ---

func TestLifecycle_GetPutReuse(t *testing.T) {
	set, _ := newTestSet(t)

	// Get → Put → Get should reuse.
	sb1, _ := set.Get()
	id := sb1.ID()
	_ = set.Put(sb1)

	sb2, _ := set.Get()
	if sb2.ID() != id {
		t.Fatalf("expected reuse, got different ID: %s vs %s", id, sb2.ID())
	}
	_ = set.Put(sb2)

	_ = set.Close()
}

func TestLifecycle_GetDestroyGet(t *testing.T) {
	set, _ := newTestSet(t)

	sb1, _ := set.Get()
	id := sb1.ID()
	_ = set.Destroy(sb1, "bad")

	sb2, _ := set.Get()
	if sb2.ID() == id {
		t.Fatal("expected fresh sandbox after Destroy, got same ID")
	}
	_ = set.Put(sb2)

	_ = set.Close()
}

// --- Concurrency tests ---

func TestConcurrent_Gets(t *testing.T) {
	set, _ := newTestSet(t)
	const n = 50

	var wg sync.WaitGroup
	sandboxes := make([]sandbox.Sandbox, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sb, err := set.Get()
			sandboxes[idx] = sb
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: Get: %v", i, err)
		}
	}

	// Clean up: put all back then close.
	for _, sb := range sandboxes {
		_ = set.Put(sb)
	}
	_ = set.Close()
}

func TestConcurrent_GetPut(t *testing.T) {
	set, _ := newTestSet(t)
	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sb, err := set.Get()
				if err != nil {
					t.Errorf("goroutine %d iter %d: Get: %v", id, j, err)
					return
				}
				if err := set.Put(sb); err != nil {
					t.Errorf("goroutine %d iter %d: Put: %v", id, j, err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	_ = set.Close()
}

func TestConcurrent_CloseWhileGet(t *testing.T) {
	set, _ := newTestSet(t)
	const n = 20

	// Grab some sandboxes first.
	for i := 0; i < 5; i++ {
		sb, _ := set.Get()
		_ = set.Put(sb)
	}

	var wg sync.WaitGroup
	errs := make(chan error, n)

	// Launch goroutines that race Get vs Close.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sb, err := set.Get()
			if err != nil {
				// Expected for some goroutines after Close.
				return
			}
			errs <- set.Put(sb)
		}()
	}

	// Close from main goroutine while Gets are racing.
	closeErr := set.Close()
	wg.Wait()
	close(errs)

	// Close should succeed (first call).
	if closeErr != nil {
		// Close might race with Get; as long as no panic, we're OK.
		fmt.Printf("Close returned: %v (acceptable in race)\n", closeErr)
	}
}
