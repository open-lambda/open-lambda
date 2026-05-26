//go:build integration

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
	"github.com/open-lambda/open-lambda/go/worker/sandboxset"
)

// newDockerSet creates a SandboxSet backed by a real DockerPool.
// Requires Docker daemon running and the ol-min image available.
func newDockerSet(t *testing.T) sandboxset.SandboxSet {
	t.Helper()

	tmpDir := t.TempDir()
	workerDir := filepath.Join(tmpDir, "worker")
	pkgsDir := filepath.Join(tmpDir, "packages")
	codeDir := filepath.Join(tmpDir, "code")

	for _, d := range []string{workerDir, pkgsDir, codeDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	common.Conf = &common.Config{
		Worker_dir: workerDir,
		Pkgs_dir:   pkgsDir,
		Sandbox:    "docker",
		Docker: common.DockerConfig{
			Base_image: "ol-min",
		},
		Limits: common.LimitsConfig{
			Procs:       10,
			Mem_mb:      50,
			CPU_percent: 100,
			Swappiness:  0,
			Runtime_sec: 30,
		},
	}

	pool, err := sandbox.NewDockerPool("", nil)
	if err != nil {
		t.Fatalf("NewDockerPool: %v (is Docker running? is ol-min image built?)", err)
	}

	scratchDirs, err := common.NewDirMaker("scratch", common.STORE_REGULAR)
	if err != nil {
		t.Fatal(err)
	}

	set := sandboxset.New(&sandboxset.Config{
		Pool:        pool,
		IsLeaf:      true,
		CodeDir:     codeDir,
		ScratchDirs: scratchDirs,
	})

	t.Cleanup(func() {
		_ = set.Close()
		pool.Cleanup()
	})

	return set
}

func TestIntegration_GetCreatesRealContainer(t *testing.T) {
	set := newDockerSet(t)

	ref, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("GetOrCreateUnpaused: %v", err)
	}
	sb := ref.Sandbox()

	if sb.ID() == "" {
		t.Fatal("expected non-empty sandbox ID")
	}
	t.Logf("created real container: ID=%s", sb.ID())
	t.Logf("debug: %s", sb.DebugString())

	// Caller owns lifecycle: destroy the sandbox, mark the ref dead, release.
	ref.Sandbox().Destroy("test cleanup")
	ref.MarkDead()
	ref.Put()
}

func TestIntegration_PutPausesAndReuses(t *testing.T) {
	set := newDockerSet(t)

	// Get a sandbox, record its ID, put it back
	ref1, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("first GetOrCreateUnpaused: %v", err)
	}
	id1 := ref1.Sandbox().ID()
	t.Logf("first sandbox: ID=%s", id1)

	ref1.Put()

	// Get again — should reuse the same container (unpaused from paused state)
	ref2, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("second GetOrCreateUnpaused: %v", err)
	}
	id2 := ref2.Sandbox().ID()
	t.Logf("second sandbox: ID=%s", id2)

	if id2 != id1 {
		t.Fatalf("expected reuse (same ID %s), got new container %s", id1, id2)
	}

	ref2.Sandbox().Destroy("test cleanup")
	ref2.MarkDead()
	ref2.Put()
}

func TestIntegration_MarkDeadGetsNew(t *testing.T) {
	set := newDockerSet(t)

	ref1, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("first GetOrCreateUnpaused: %v", err)
	}
	id1 := ref1.Sandbox().ID()
	t.Logf("first sandbox: ID=%s", id1)

	// Caller-owned destroy + MarkDead + Put releases the slot without a sandbox.
	ref1.Sandbox().Destroy("test: simulate handler failure")
	ref1.MarkDead()
	ref1.Put()

	// Get again — must be a different container since the slot is empty
	ref2, err := set.GetOrCreateUnpaused()
	if err != nil {
		t.Fatalf("second GetOrCreateUnpaused: %v", err)
	}
	id2 := ref2.Sandbox().ID()
	t.Logf("second sandbox: ID=%s", id2)

	if id2 == id1 {
		t.Fatal("expected new container after MarkDead, got same ID")
	}

	ref2.Sandbox().Destroy("test cleanup")
	ref2.MarkDead()
	ref2.Put()
}

func TestIntegration_CloseDestroysAll(t *testing.T) {
	set := newDockerSet(t)

	// Create multiple sandboxes
	refs := make([]*sandboxset.SandboxRef, 3)
	for i := range refs {
		ref, err := set.GetOrCreateUnpaused()
		if err != nil {
			t.Fatalf("GetOrCreateUnpaused[%d]: %v", i, err)
		}
		refs[i] = ref
		t.Logf("sandbox[%d]: ID=%s", i, ref.Sandbox().ID())
	}

	// Put one back to idle so Close covers the idle path.
	refs[2].Put()

	// Close destroys the idle sandbox; in-use refs are destroyed by put()
	// when their holders return them below.
	if err := set.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Put after Close routes through put()'s closed branch, which destroys
	// the sandbox. The caller does not Destroy here.
	for i := 0; i < 2; i++ {
		refs[i].Put()
	}

	// Verify set is closed — further Gets should fail
	_, err := set.GetOrCreateUnpaused()
	if err == nil {
		t.Fatal("expected error after Close")
	}
}