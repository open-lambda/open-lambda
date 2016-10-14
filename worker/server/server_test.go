package server

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
)

var server *Server

func init() {
	server = RunServer()
}

func RunServer() *Server {
	conf, err := config.ParseConfig(os.Getenv("WORKER_CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(conf.Registry)
	log.Printf("Set skip_pull_existing = true\n")
	conf.Skip_pull_existing = true

	server, err := NewServer(conf)
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

func last_count(img string) int {
	logs, err := server.handlers.Get(img).Sandbox().Logs()
	if err != nil {
		log.Fatal(err.Error())
	}
	lines := strings.Split(logs, "\n")
	for i := range lines {
		line := lines[len(lines)-i-1]
		parts := strings.Split(line, "=")
		if parts[0] == "counter" {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				return n
			} else {
				panic("not an int: " + parts[1])
			}
		}
	}
	return 0
}

// thread_counter starts a backup thread that runs forever,
// incrementing a counter between 10ms sleeps.  If pausing works, the
// counter won't tick many times between requests, even if wait
// between them.
func TestThreadPausing(t *testing.T) {
	img := "thread_counter"
	testReq(img, "null")
	count1 := last_count(img)
	time.Sleep(100 * time.Millisecond)
	count2 := last_count(img)
	if count1 <= 0 {
		log.Fatal("count1 isn't positive")
	}
	if count2 != count1 {
		log.Fatal("count1 != count2")
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

func BenchmarkEchoParallel(b *testing.B) {
	values := []string{
		"{\"one\": 1}",
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			recv, err := testReq("echo", values[0])
			if err != nil {
				b.Fatal(err)
			}
			if recv != values[0] {
				b.Fatalf("Sent '%v' to echo but got back '%v'\n", values[0], recv)
			}
		}
	})
}
