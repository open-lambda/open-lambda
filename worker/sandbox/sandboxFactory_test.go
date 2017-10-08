package sandbox

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/worker/config"
)

func getConf() *config.Config {
	conf, err := config.ParseConfig(os.Getenv("WORKER_CONFIG"))
	if err != nil {
		log.Fatal(err)
	}
	return conf
}

func getClient() *docker.Client {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func dockerExec(client *docker.Client, cid string, cmd []string) error {
	createOpt := docker.CreateExecOptions{Cmd: cmd, Container: cid}
	startOpt := docker.StartExecOptions{}

	if exec, err := client.CreateExec(createOpt); err != nil {
		return err
	} else if err := client.StartExec(exec.ID, startOpt); err != nil {
		return err
	}
	return nil
}

func TestWriteToDirs(t *testing.T) {
	config := getConf()
	client := getClient()

	dockerSbFactory, err := NewDockerSBFactory(config)
	if err != nil {
		t.Fatalf("cannot create docker sandbox factory")
	}

	handlerDir := "/tmp/.handlerDir"
	sandboxDir := "/tmp/.sandboxDir"

	if err = os.RemoveAll(handlerDir); err != nil {
		t.Fatalf("cannot remove handler directory: ", err)
	}
	if err = os.MkdirAll(handlerDir, 0777); err != nil {
		t.Fatalf("cannot create handler directory: ", err)
	}
	if err = os.RemoveAll(sandboxDir); err != nil {
		t.Fatalf("cannot remove sandbox directory: ", err)
	}
	if err = os.MkdirAll(sandboxDir, 0777); err != nil {
		t.Fatalf("cannot create sandbox directory: ", err)
	}

	if sandbox, err := dockerSbFactory.Create(handlerDir, sandboxDir, "", ""); err != nil {
		t.Fatalf("fail to create sandbox: ", err)
	} else if err := sandbox.Start(); err != nil {
		t.Fatalf("fail to start sandbox: ", err)
	} else {
		dockerSandbox := sandbox.(*DockerSandbox)
		cid := dockerSandbox.container.ID
		if err = dockerExec(client, cid, []string{"touch", "/handler/should_fail"}); err != nil {
			t.Fatalf("fail to execute command in container: ", err)
		}
		if err = dockerExec(client, cid, []string{"touch", "/host/should_succeed"}); err != nil {
			t.Fatalf("fail to execute command in container: ", err)
		}

		time.Sleep(time.Millisecond * 500)
		if _, err := os.Stat("/tmp/.handlerDir/should_fail"); os.IsExist(err) {
			t.Fatalf("should not be able to write to handler directory: ", err)
		}
		if _, err := os.Stat(fmt.Sprintf("/tmp/.sandboxDir/%s/should_succeed", sandbox.ID())); os.IsNotExist(err) {
			t.Fatalf("should be able to write to sandbox directory: ", err)
		}
		time.Sleep(time.Second)
	}
}
