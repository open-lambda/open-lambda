package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"testing"
)

func RunServer() *Server {
	server, err := NewServer("", "", "")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		http.HandleFunc("/runLambda/", server.RunLambda)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	return server
}

func testReq() error {
	url := "http://localhost:8080/runLambda/pausable-start-timer"
	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("RESP: %v\n", string(body))
	return nil
}

func TestPull(t *testing.T) {
	server := RunServer()

	for i := 1; i <= 10; i++ {
		err := testReq()
		if err != nil {
			t.Error(err)
		}
	}
	server.Manager().Dump()
	server.Dump()
}
