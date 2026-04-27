package tests

import (
	"errors"
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
	set := sandboxset.New(&sandboxset.Config{
		Pool:        pool,
		CodeDir:     tmpDir + "/code",
		ScratchDirs: scratchDirs,
	})
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

	ref1.Put()

	ref2, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("second GetOrCreateUnpaused: %v", err)
	}
	if ref2.Sandbox().ID() != id {
		t.Fatalf("expected reuse (ID %s), got new (ID %s)", id, ref2.Sandbox().ID())
	}
	ref2.Put()
}

// TestMarkDead_NewSandboxOnNextGet verifies that after MarkDead+Put the slot
// is empty, and the next GetOrCreateUnpaused creates a fresh sandbox in it
// (pool size stays 1, total created sandboxes grows to 2).
func TestMarkDead_NewSandboxOnNextGet(t *testing.T) {
	set, pool := newTestSet(t)
	defer set.Close()

	ref1, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	id1 := ref1.Sandbox().ID()

	ref1.MarkDead()
	ref1.Put()

	ref2, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("second GetOrCreateUnpaused: %v", err)
	}
	if ref2.Sandbox().ID() == id1 {
		t.Fatal("expected a new sandbox after MarkDead, got the same ID")
	}
	if n := len(pool.CreatedSandboxes()); n != 2 {
		t.Fatalf("expected 2 created sandboxes total, got %d", n)
	}
	ref2.Put()
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

// TestPut_Twice_Panics verifies the double-Put guard: the second Put on a
// ref that's already been returned must panic rather than silently corrupt.
func TestPut_Twice_Panics(t *testing.T) {
	set, _ := newTestSet(t)
	defer set.Close()

	ref, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	ref.Put()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on double Put")
		}
	}()
	ref.Put()
}

// TestPut_PauseFailure_DestroysSandbox verifies that when Pause fails inside
// put(), the orphaned sandbox is destroyed by the set rather than leaked.
func TestPut_PauseFailure_DestroysSandbox(t *testing.T) {
	set, pool := newTestSet(t)
	defer set.Close()

	ref, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	sb := pool.CreatedSandboxes()[0]
	sb.PauseErr = errors.New("simulated pause failure")

	ref.Put()

	if !sb.IsDestroyed() {
		t.Fatal("expected sandbox to be destroyed after Pause failure in Put")
	}
}

// TestClose_DestroysIdleSandbox verifies Close destroys idle sandboxes
// (which the set is the only holder of).
func TestClose_DestroysIdleSandbox(t *testing.T) {
	set, pool := newTestSet(t)

	ref, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	sb := pool.CreatedSandboxes()[0]
	ref.Put() // ref now idle in the pool

	if err := set.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !sb.IsDestroyed() {
		t.Fatal("expected idle sandbox to be destroyed by Close")
	}
}

// TestPut_AfterClose_DestroysSandbox verifies that a Put arriving after Close
// destroys the returned sandbox (the set is the only remaining holder).
func TestPut_AfterClose_DestroysSandbox(t *testing.T) {
	set, pool := newTestSet(t)

	ref, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	sb := pool.CreatedSandboxes()[0]

	if err := set.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ref.Put()

	if !sb.IsDestroyed() {
		t.Fatal("expected sandbox to be destroyed by Put after Close")
	}
}

// TestMarkDead_AfterPut_Panics verifies MarkDead is rejected on a ref that
// is no longer held.
func TestMarkDead_AfterPut_Panics(t *testing.T) {
	set, _ := newTestSet(t)
	defer set.Close()

	ref, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	ref.Put()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on MarkDead after Put")
		}
	}()
	ref.MarkDead()
}
