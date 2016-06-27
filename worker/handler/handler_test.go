package handler

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/sandbox"
)

type MockSandboxManager struct{}

func (MockSandboxManager) Create(name string) sandbox.Sandbox {
	return nil
}

func NewDockerManager() (manager *sandbox.DockerManager) {
	reg := os.Getenv("TEST_REGISTRY")
	log.Printf("Use registry %v", reg)
	components := strings.Split(reg, ":")
	return sandbox.NewDockerManager(components[0], components[1])
}

func TestHandlerLookupSame(t *testing.T) {
	handlers := NewHandlerSet(HandlerSetOpts{Cm: MockSandboxManager{}})
	a1 := handlers.Get("a")
	a2 := handlers.Get("a")
	if a1 != a2 {
		t.Fatal("got different handlers for same name")
	}
}

func TestHandlerLookupDiff(t *testing.T) {
	handlers := NewHandlerSet(HandlerSetOpts{Cm: MockSandboxManager{}})
	a := handlers.Get("a")
	b := handlers.Get("b")
	if a == b {
		t.Fatal("got same handlers for different name")
	}
}

func TestHandlerHandlerPull(t *testing.T) {
	cm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Cm: cm})
	name := "nonlocal"

	// sandboxs should initially not be pulled
	exists, err := cm.DockerImageExists(name)
	if err != nil {
		t.Fatal(err.Error())
	}
	if exists {
		t.Fatalf("make sure %s is not pulled before test", name)
	}

	h := handlers.Get(name)

	// Get SHOULD NOT trigger pull
	exists, err = cm.DockerImageExists(name)
	if err != nil {
		t.Fatal(err.Error())
	}
	if exists {
		t.Fatalf("Get should not pull %s", name)
	}

	h.RunStart()

	// Run SHOULD trigger pull
	exists, err = cm.DockerImageExists(name)
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
	cm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Cm: cm, Lru: lru})
	h := handlers.Get("hello2")

	h.RunStart()
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
	cm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Cm: cm, Lru: lru})
	h := handlers.Get("hello2")
	count := 10

	for i := 0; i < count; i++ {
		h.RunStart()
		s := GetState(t, h)
		if !(s == state.Running) {
			t.Fatalf("Unexpected state: %v", s.String())
		}
	}

	for i := 0; i < count; i++ {
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
	cm := NewDockerManager()
	handlers := NewHandlerSet(HandlerSetOpts{Cm: cm, Lru: lru})
	h := handlers.Get("hello2")
	h.RunStart()
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
