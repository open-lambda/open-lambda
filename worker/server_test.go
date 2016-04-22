package main

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
)

func RunServer() *Server {
	server, err := NewServer("", "", "")
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/runLambda/", server.RunLambda)
	s := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		log.Fatal(s.Serve(listener))
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
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func TestPull(t *testing.T) {
	RunServer()
	err := testReq()
	if err != nil {
		t.Error(err)
	}
}
