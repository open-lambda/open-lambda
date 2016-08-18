package main

import (
	"fmt"
	"io/ioutil"
	"log"

	r "github.com/open-lambda/worker/registry"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	cluster := []string{"127.0.0.1:28015"}

	spull := r.InitPullClient(cluster)
	fmt.Println("Running pullclient...")
	files := spull.Pull("test")
	handler := files.Handler
	pb := sfiles.PB

	err = ioutil.WriteFile("handler.go", handler, 0644)
	check(err)
}
