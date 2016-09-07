// +build pushserver

package main

import (
	"log"
	"os"
	"strconv"

	r "github.com/open-lambda/open-lambda/registry/src"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: pushserver <port> <cluster_ip1> <cluster_ip2>...")
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	pushs := r.InitPushServer(port, os.Args[2:])
	pushs.Run()
}
