package main

import (
	"log"
	"os"

	r "github.com/open-lambda/open-lambda/registry/src"
)

func main() {
	if len(os.Args) == 1 {
		log.Fatal("Usage: pushserver <cluster_ip1> <cluster_ip2>...")
	}

	cluster := make([]string, 0)
	for _, ip := range os.Args {
		cluster = append(cluster, ip)
	}

	pushs := r.InitPushServer(cluster)
	pushs.Run()
}
