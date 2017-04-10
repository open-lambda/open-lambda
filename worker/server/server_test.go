package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

var server *Server
var docker_client *docker.Client

func init() {
	server = RunServer()
	var err error
	docker_client, err = docker.NewClientFromEnv()
	if err != nil {
		log.Fatal("failed to get docker client: ", err)
	}
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

func kill() {
	filters := map[string][]string{
		"label": {fmt.Sprintf("%s=%s", dockerutil.DOCKER_LABEL_CLUSTER, server.config.Cluster_name)},
	}
	containers, err := docker_client.ListContainers(docker.ListContainersOptions{Filters: filters})
	if err != nil {
		log.Fatal("failed to get docker container list: ", err)
	}

	for _, container := range containers {
		dockerutil.SafeRemove(docker_client, container.ID)
	}
}

func TestMain(m *testing.M) {
	ret_val := m.Run()
	fmt.Printf("\n========Cleaning========\n")
	kill()
	fmt.Printf("========================\n\n")
	os.Exit(ret_val)
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

func TestInstall(t *testing.T) {
	recv, err := testReq("install", "{}")
	if err != nil {
		t.Fatal(err)
	}
	expected := "\"imported\""
	if recv != expected {
		t.Fatalf("Expected 'imported' from Install but got back '%v'\n", recv)
	}
}

func last_count(img string) (int, error) {
	h := server.handlers.Get(img)

	logs, err := h.Sandbox().Logs()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(logs, "\n")
	for i := range lines {
		line := lines[len(lines)-i-1]
		parts := strings.Split(line, "=")
		if parts[0] == "counter" {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				return n, nil
			} else {
				return 0, err
			}
		}
	}

	return 0, nil
}

// thread_counter starts a backup thread that runs forever,
// incrementing a counter between 10ms sleeps.  If pausing works, the
// counter won't tick many times between requests, even if wait
// between them.
func TestThreadPausing(t *testing.T) {
	img := "thread_counter"
	testReq(img, "null")

	count1, err := last_count(img)
	if err != nil {
		t.Fatalf("Failed to get first count with: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	count2, err := last_count(img)
	if err != nil {
		t.Fatalf("Failed to get second count with: %v", err)
	}

	if count1 <= 0 {
		t.Fatalf("count1 isn't positive (%d) - logs working?", count1)
	}
	if count2 != count1 {
		t.Fatalf("count1 (%d) != count2 (%d)", count1, count2)
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
