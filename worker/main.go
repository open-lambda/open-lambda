package main

import (
	"github.com/open-lambda/open-lambda/worker/server"
	"log"
	"os"
)

func main() {
	// parse config file
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <json-config>\n", os.Args[0])
	}

	server.Main(os.Args[1])
}
