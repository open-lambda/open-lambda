package main

import (
	"testing"
	"log"
	"fmt"
	"strings"
	"time"
	"net/http"
)

func RunServer() {
	server,err := NewServer("", "", "")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/runLambda/", server.RunLambda)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func TestPull(t *testing.T) {
	go RunServer()

	time.Sleep(1000 * time.Millisecond) // TODO

	url := "http://localhost:8080/runLambda/hello"
	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	// TODO: check resp
	fmt.Printf("RESP: %v, %v\n", resp, err)
}
