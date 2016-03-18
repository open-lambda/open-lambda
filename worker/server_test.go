package main

import (
	"testing"
	"log"
	"fmt"
	"strings"
	"net/http"
)

func RunServer() {
	server := Server{}
	http.HandleFunc("/runLambda/", server.RunLambda)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func TestPull(t *testing.T) {
	go RunServer()

	url := "http://localhost:8080/runLambda/hello"
	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	// TODO: check resp
	fmt.Printf("%v, %v\n", resp, err)
}
