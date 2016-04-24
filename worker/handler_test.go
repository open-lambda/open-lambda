package main

import (
	"testing"
)

func TestHandlerLookupSame(t *testing.T) {
	handlers := NewHandlerSet(nil)
	a1 := handlers.Get("a")
	a2 := handlers.Get("a")
	if a1 != a2 {
		t.Fatal("got different handlers for same name")
	}
}

func TestHandlerLookupDiff(t *testing.T) {
	handlers := NewHandlerSet(nil)
	a := handlers.Get("a")
	b := handlers.Get("b")
	if a == b {
		t.Fatal("got same handlers for different name")
	}
}

func TestHandlerHandlerInit(t *testing.T) {
	cm := NewContainerManager("localhost", "5000")
	handlers := NewHandlerSet(cm)
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
