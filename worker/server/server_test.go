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

	"github.com/open-lambda/open-lambda/worker/config"
	sbmanager "github.com/open-lambda/open-lambda/worker/sandbox-manager"
	docker "github.com/fsouza/go-dockerclient"
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
        containers, err := docker_client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		log.Fatal("failed to get docker container list: ", err)
	}

	for _, container := range containers {
            if container.Labels[sbmanager.DOCKER_LABEL_CLUSTER] == server.config.Cluster_name {
                cid := container.ID
                typ := server.config.Cluster_name

	        container_insp, err := docker_client.InspectContainer(cid)
                if err != nil {
		    log.Fatalf("failed to get inspect docker container ID %v: ", cid, err)
	        }

                if container_insp.State.Paused {
		    fmt.Printf("Unpause container %v (%s)\n", cid, typ)
		    if err := docker_client.UnpauseContainer(cid); err != nil {
			fmt.Printf("%s\n", err.Error())
			fmt.Printf("Failed to unpause container %v (%s).  May require manual cleanup.\n", cid, typ)
		    }
		}

		fmt.Printf("Kill container %v (%s)\n", cid, typ)
		opts := docker.KillContainerOptions{ID: cid}
		if err := docker_client.KillContainer(opts); err != nil {
		    fmt.Printf("%s\n", err.Error())
		    fmt.Printf("Failed to kill container %v (%s).  May require manual cleanup.\n", cid, typ)
		}
            }
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
