package main

import (
	"fmt"
	r "github.com/open-lambda/open-lambda/worker/registry"
)

func main() {
	saddr := fmt.Sprintf("127.0.0.1:%d", r.SPORT)
	pushc := r.InitPushClient(saddr)
	fmt.Println("Pushing from client...")
	pushc.PushFiles("test", "handler.in", r.HANDLER)
}
