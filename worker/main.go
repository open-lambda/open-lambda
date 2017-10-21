package main

import (
	"github.com/open-lambda/open-lambda/worker/server"
	"log"
	"os"
	"runtime"
	"sanbox"
)

func main() {
	// parse config file
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <json-config>\n", os.Args[0])
	}

	runtime.GOMAXPROCS(runtime.NumCPU() + sandbox.numUnmountWorkers)

	server.Main(os.Args[1])
}
