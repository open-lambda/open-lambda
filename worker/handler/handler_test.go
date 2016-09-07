package handler

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/sandbox"
)

func NewDockerManager() (manager *sandbox.DockerManager) {
	conf, err := config.ParseConfig(os.Getenv("WORKER_CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Set skip_pull_existing = true\n")
	conf.Skip_pull_existing = true

	return sandbox.NewDockerManager(conf)
}

func TestHandlerLookupSame(t *testing.T) {
	sm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Sm: sm})
	a1 := handlers.Get("a")
	a2 := handlers.Get("a")
	if a1 != a2 {
		t.Fatal("got different handlers for same name")
	}
}

func TestHandlerLookupDiff(t *testing.T) {
	sm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Sm: sm})
	a := handlers.Get("a")
	b := handlers.Get("b")
	if a == b {
		t.Fatal("got same handlers for different name")
	}
}

func TestHandlerHandlerPull(t *testing.T) {
	sm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Sm: sm})
	name := "nonlocal"

	exists, err := sm.DockerImageExists(name)
	if err != nil {
		t.Fatal(err.Error())
	}
	if exists {
		t.Fatalf("make sure %s is not pulled before test", name)
	}

	h := handlers.Get(name)

	// Get SHOULD NOT trigger pull
	exists, err = sm.DockerImageExists(name)
	if err != nil {
		t.Fatal(err.Error())
	}
	if exists {
		t.Fatalf("Get should not pull %s", name)
	}

	_, err = h.RunStart()
	if err != nil {
		t.Fatalf("RunStart failed with %v", err.Error())
	}

	// Run SHOULD trigger pull
	exists, err = sm.DockerImageExists(name)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !exists {
		t.Fatalf("Lambda %s not started by run", name)
	}
}

func GetState(t *testing.T, h *Handler) state.HandlerState {
	state, err := h.sandbox.State()
	if err != nil {
		t.Fatalf("Could not get state for %v", h.name)
	}
	return state
}

func TestHandlerRunCountOne(t *testing.T) {
	lru := NewHandlerLRU(1)
	sm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Sm: sm, Lru: lru})
	h := handlers.Get("hello2")

	_, err := h.RunStart()
	if err != nil {
		t.Fatalf("RunStart failed with %v", err.Error())
	}
	s := GetState(t, h)
	if !(s == state.Running) {
		t.Fatalf("Unexpected state: %v", s.String())
	}

	h.RunFinish()
	s = GetState(t, h)
	if !(s == state.Paused) {
		t.Fatalf("Unexpected state(2): %v", s.String())
	}
}

func TestHandlerRunCountMany(t *testing.T) {
	lru := NewHandlerLRU(1)
	sm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Sm: sm, Lru: lru})
	h := handlers.Get("hello2")
	count := 10

	for i := 0; i < count; i++ {
		log.Printf("Starting %v\n", i+1)
		_, err := h.RunStart()
		if err != nil {
			t.Fatalf("RunStart failed with %v", err.Error())
		}
		s := GetState(t, h)
		if !(s == state.Running) {
			t.Fatalf("Unexpected state: %v", s.String())
		}
	}

	for i := 0; i < count; i++ {
		log.Printf("Finishing %v\n", i+1)
		h.RunFinish()
		s := GetState(t, h)
		if i == count-1 {
			if !(s == state.Paused) {
				t.Fatalf("Unexpected state: %v", s.String())
			}
		} else {
			if !(s == state.Running) {
				t.Fatalf("Unexpected state: %v", s.String())
			}
		}
	}
}

func TestHandlerEvict(t *testing.T) {
	lru := NewHandlerLRU(0)
	sm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Sm: sm, Lru: lru})
	h := handlers.Get("hello2")
	_, err := h.RunStart()
	if err != nil {
		t.Fatalf("RunStart failed with %v", err.Error())
	}
	h.RunFinish()
	s := GetState(t, h)

	// wait up to 5 seconds for evictor to evict
	max_tries := 500
	for tries := 1; ; tries++ {
		s = GetState(t, h)
		if !(s == state.Running) {
			return
		} else if tries == max_tries {
			t.Fatalf("Unexpected state: %v", s.String())
		}
		time.Sleep(100 * time.Millisecond)

	}
}
