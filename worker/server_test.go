package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
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

// thread_counter starts a backup thread that runs forever,
// incrementing a counter between 10ms sleeps.  If pausing works, the
// counter won't tick many times between requests, even if wait
// between them.
func TestThreadPausing(t *testing.T) {
	img := "thread_counter"
	before_str, _ := testReq(img, "null")
	time.Sleep(100 * time.Millisecond)
	after_str, _ := testReq(img, "null")
	before, _ := strconv.Atoi(before_str)
	after, _ := strconv.Atoi(after_str)
	if after-before > 2 {
		t.Errorf("Background thread run between requests, ticking %v times\n", (after - before))
	}
}

func BenchmarkEcho(b *testing.B) {
	values := []string{
		"{\"one\": 1}",
	}
	for i := 0; i < b.N; i++ {
		recv, err := testReq("echo", values[0])
		if err != nil {
			b.Fatal(err)
		}
		if recv != values[0] {
			b.Fatalf("Sent '%v' to echo but got back '%v'\n", values[0], recv)
		}
	}
}

// Run benchmark at each test, and print nicely
func TestFancyBenchmark(t *testing.T) {
	br := testing.Benchmark(BenchmarkEcho)
	us := br.NsPerOp() / (1000)
	fmt.Printf("\t--- Echo Benchmark Report ---\n")
	fmt.Printf("\tTiming: %vus per lambda req/rsp\n", us)
	fmt.Printf("\tMemory: %v\n", br.MemString())
}
