package main

import (
	docker "github.com/fsouza/go-dockerclient"
	"testing"
	"time"
)

func TestHandlerLookupSame(t *testing.T) {
	handlers := NewHandlerSet(HandlerSetOpts{})
	a1 := handlers.Get("a")
	a2 := handlers.Get("a")
	if a1 != a2 {
		t.Fatal("got different handlers for same name")
	}
}

func TestHandlerLookupDiff(t *testing.T) {
	handlers := NewHandlerSet(HandlerSetOpts{})
	a := handlers.Get("a")
	b := handlers.Get("b")
	if a == b {
		t.Fatal("got same handlers for different name")
	}
}

func TestHandlerHandlerPull(t *testing.T) {
	cm := NewContainerManager("localhost", "5000")
	handlers := NewHandlerSet(HandlerSetOpts{cm: cm})
	name := "nonlocal"

	// containers should initially not be pulled
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

func GetState(t *testing.T, cm *ContainerManager, img string) docker.State {
	container, err := cm.DockerInspect(img)
	if err != nil {
		t.Fatalf("Could not inspect '%v'", img)
	}
	return container.State
}

func TestHandlerRunCountOne(t *testing.T) {
	lru := NewHandlerLRU(1)
	cm := NewContainerManager("localhost", "5000")
	handlers := NewHandlerSet(HandlerSetOpts{cm: cm, lru: lru})
	h := handlers.Get("hello")

	h.RunStart()
	s := GetState(t, cm, "hello")
	if !(s.Running && !s.Paused) {
		t.Fatalf("Unexpected state: %v", s.StateString())
	}

	h.RunFinish()
	s = GetState(t, cm, "hello")
	if !(s.Running && s.Paused) {
		t.Fatalf("Unexpected state: %v", s.StateString())
	}
}

func TestHandlerRunCountMany(t *testing.T) {
	lru := NewHandlerLRU(1)
	cm := NewContainerManager("localhost", "5000")
	handlers := NewHandlerSet(HandlerSetOpts{cm: cm, lru: lru})
	h := handlers.Get("hello")
	count := 10

	for i := 0; i < count; i++ {
		h.RunStart()
		s := GetState(t, cm, "hello")
		if !(s.Running && !s.Paused) {
			t.Fatalf("Unexpected state: %v", s.StateString())
		}
	}

	for i := 0; i < count; i++ {
		h.RunFinish()
		s := GetState(t, cm, "hello")
		if i == count-1 {
			if !(s.Running && s.Paused) {
				t.Fatalf("Unexpected state: %v", s.StateString())
			}
		} else {
			if !(s.Running && !s.Paused) {
				t.Fatalf("Unexpected state: %v", s.StateString())
			}
		}
	}
}

func TestHandlerEvict(t *testing.T) {
	lru := NewHandlerLRU(0)
	cm := NewContainerManager("localhost", "5000")
	handlers := NewHandlerSet(HandlerSetOpts{cm: cm, lru: lru})
	h := handlers.Get("hello")
	h.RunStart()
	h.RunFinish()
	s := GetState(t, cm, "hello")

	// wait up to 5 seconds for evictor to evict
	max_tries := 500
	for tries := 1; ; tries++ {
		s = GetState(t, cm, "hello")
		if !s.Running {
			return
		} else if tries == max_tries {
			t.Fatalf("Unexpected state: %v", s.StateString())
		}
		time.Sleep(100 * time.Millisecond)

	}
}
