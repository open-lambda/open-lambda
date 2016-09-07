// +build regpush

package main

import (
	"log"
	"os"

	r "github.com/open-lambda/open-lambda/registry/src"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatal("Usage: pushserver <server_ip> <name> <file>")
	}

	pushc := r.InitPushClient(os.Args[1])
	pushc.PushFiles(os.Args[2], os.Args[3], r.HANDLER)
}
