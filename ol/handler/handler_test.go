package handler

import (
	"log"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/ol/handler/state"
	"github.com/open-lambda/open-lambda/ol/sandbox/dockerutil"
)

func getClient() *docker.Client {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func TestHandlerLookupSame(t *testing.T) {
	handlers, err := NewHandlerSet(getConf(), NewHandlerLRU(0))
	if err != nil {
		t.Fatal(err.Error())
	}

	a1 := handlers.Get("a")
	a2 := handlers.Get("a")
	if a1 != a2 {
		t.Fatal("got different handlers for same name")
	}
}

func TestHandlerLookupDiff(t *testing.T) {
	handlers, err := NewHandlerSet(getConf(), NewHandlerLRU(0))
	if err != nil {
		t.Fatal(err.Error())
	}
	a := handlers.Get("a")
	b := handlers.Get("b")
	if a == b {
		t.Fatal("got same handlers for different name")
	}
}

func TestHandlerHandlerPull(t *testing.T) {
	t.Skip("TestHandlerHandlerPull does not work with local registry mode")

	handlers, err := NewHandlerSet(getConf(), NewHandlerLRU(0))
	if err != nil {
		t.Fatal(err.Error())
	}
	name := "nonlocal"

	exists, err := dockerutil.ImageExists(getClient(), name)
	if err != nil {
		t.Fatal(err.Error())
	}
	if exists {
		t.Fatalf("make sure %s is not pulled before test", name)
	}

	h := handlers.Get(name)

	// Get SHOULD NOT trigger pull
	exists, err = dockerutil.ImageExists(getClient(), name)
	if err != nil {
		t.Fatal(err.Error())
	}
	if exists {
		t.Fatalf("Get should not pull %s", name)
	}

	_, err = h.RunStart()
	if err != nil {
		t.Fatalf("RunStart failed with: %v", err.Error())
	}

	// Run SHOULD trigger pull
	exists, err = dockerutil.ImageExists(getClient(), name)
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
	handlers, err := NewHandlerSet(getConf(), lru)
	if err != nil {
		t.Fatal(err.Error())
	}
	h := handlers.Get("hello2")

	_, err = h.RunStart()
	if err != nil {
		t.Fatalf("RunStart failed with: %v", err.Error())
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
	handlers, err := NewHandlerSet(getConf(), lru)
	if err != nil {
		t.Fatal(err.Error())
	}
	h := handlers.Get("hello2")
	count := 10

	for i := 0; i < count; i++ {
		log.Printf("Starting %v\n", i+1)
		_, err := h.RunStart()
		if err != nil {
			t.Fatalf("RunStart failed with: %v", err.Error())
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
	handlers, err := NewHandlerSet(getConf(), lru)
	if err != nil {
		t.Fatal(err.Error())
	}
	h := handlers.Get("hello2")
	_, err = h.RunStart()
	if err != nil {
		t.Fatalf("RunStart failed with: %v", err.Error())
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
