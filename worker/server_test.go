package main

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
)

var server *Server

func init() {
	server = RunServer()
}

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

func testReq(lambda_name string, post string) (string, error) {
	url := "http://localhost:8080/runLambda/" + lambda_name
	req, err := http.NewRequest("POST", url, strings.NewReader(post))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func TestHello(t *testing.T) {
	recv, err := testReq("hello", "{}")
	if err != nil {
		t.Fatal(err)
	}
	expected := "\"hello\""
	if recv != expected {
		t.Fatalf("Expected '%v' from hello but got back '%v'\n", expected, recv)
	}
}

func TestEcho(t *testing.T) {
	values := []string{
		"{}",
		"{\"one\": 1}",
		"1",
		"\"test\"",
	}
	for _, send := range values {
		recv, err := testReq("echo", send)
		if err != nil {
			t.Fatal(err)
		}
		if recv != send {
			t.Fatalf("Sent '%v' to echo but got back '%v'\n", send, recv)
		}
	}
}

// TODO(tyler): this test needs fixing.  It passes even if the Docker
// image is not available.
func TestPull(t *testing.T) {
	_, err := testReq("pausable-start-timer", "{}")
	if err != nil {
		t.Fatal(err)
	}
}
