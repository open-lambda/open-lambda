package handler

import (
	"log"
	"os"
	"testing"

	"github.com/open-lambda/open-lambda/worker/config"
)

func getConf() *config.Config {
	conf, err := config.ParseConfig(os.Getenv("WORKER_CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	return conf
}

func TestLRU(t *testing.T) {
	lru := NewHandlerLRU(0)
	handlers, err := NewHandlerSet(getConf(), lru)
	if err != nil {
		t.Fatalf(err.Error())
	}

	a := handlers.Get("a")

	lru.Add(a)
	if lru.Len() != 1 {
		t.Fatalf("Unexpected len: %v", lru.Len())
	}
	lru.Remove(a)
	if lru.Len() != 0 {
		t.Fatalf("Unexpected len: %v", lru.Len())
	}
	lru.Add(a)
	if lru.Len() != 1 {
		t.Fatalf("Unexpected len: %v", lru.Len())
	}
}
