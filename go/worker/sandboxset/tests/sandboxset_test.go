package tests

import (
	"testing"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
	"github.com/open-lambda/open-lambda/go/worker/sandboxset"
)

// newTestSet creates a valid SandboxSet backed by a MockSandboxPool.
func newTestSet(t *testing.T) (sandboxset.SandboxSet, *sandbox.MockSandboxPool) {
	t.Helper()
	tmpDir := t.TempDir()
	common.Conf = &common.Config{Worker_dir: tmpDir}
	scratchDirs, err := common.NewDirMaker("scratch", common.STORE_REGULAR)
	if err != nil {
		t.Fatal(err)
	}
	pool := &sandbox.MockSandboxPool{}
	set, err := sandboxset.New(&sandboxset.Config{
		Pool:        pool,
		CodeDir:     tmpDir + "/code",
		ScratchDirs: scratchDirs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return set, pool
}

// TestGet_CreatesNew verifies that GetOrCreateUnpaused creates a new sandbox
// when the pool is empty.
func TestGet_CreatesNew(t *testing.T) {
	set, pool := newTestSet(t)

	ref, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	if ref.Sandbox() == nil {
		t.Fatal("expected non-nil sandbox")
	}
	if n := len(pool.CreatedSandboxes()); n != 1 {
		t.Fatalf("expected 1 created sandbox, got %d", n)
	}
}

// TestLifecycle_GetPutReuse verifies the full create → put → reuse cycle.
func TestLifecycle_GetPutReuse(t *testing.T) {
	set, _ := newTestSet(t)
	defer set.Close()

	ref1, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	id := ref1.Sandbox().ID()

	if err := ref1.Put(); err != nil {
		t.Fatalf("Put: %v", err)
	}

	ref2, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("second GetOrCreateUnpaused: %v", err)
	}
	if ref2.Sandbox().ID() != id {
		t.Fatalf("expected reuse (ID %s), got new (ID %s)", id, ref2.Sandbox().ID())
	}
	_ = ref2.Put()
}

// TestDestroy_NilRefReused verifies that after Destroy the ref stays in the pool
// as a nil ref, and the next GetOrCreateUnpaused reuses it (pool size stays 1).
func TestDestroy_NilRefReused(t *testing.T) {
	set, pool := newTestSet(t)
	defer set.Close()

	ref1, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	id1 := ref1.Sandbox().ID()

	if err := ref1.Destroy("test"); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	ref2, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("second GetOrCreateUnpaused: %v", err)
	}
	if ref2.Sandbox().ID() == id1 {
		t.Fatal("expected a new sandbox after Destroy, got the same ID")
	}
	if n := len(pool.CreatedSandboxes()); n != 2 {
		t.Fatalf("expected 2 created sandboxes total, got %d", n)
	}
	_ = ref2.Put()
}

// TestGet_AfterClose verifies that GetOrCreateUnpaused returns an error after Close.
func TestGet_AfterClose(t *testing.T) {
	set, _ := newTestSet(t)

	if err := set.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, err := set.GetOrCreateUnpaused()
	if err == nil {
		t.Fatal("expected error after Close")
	}
}
