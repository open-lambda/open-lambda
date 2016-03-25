package main

import (
	"log"
	"time"
)

const (
	nIters = 100
)

func main() {
	// TODO: registry...
	cm := NewContainerManagerWithStats("", "", "simple-run")
	simpleRunTest(cm)
	cm.DumpAndResetStats("slow-run")
	slowRunTest(cm)
	cm.DumpAndResetStats("pause-unpause")
	pauseUnpauseTest(cm)
	cm.DumpStats()
}

func simpleRunTest(cm *ContainerManager) {
	log.Println("starting simple run test...")
	// how long to run, hello, and exit?
	for i := 0; i < nIters; i++ {
		err := cm.DockerRun("hello-world", []string{"echo", "\"hello world!\""}, true)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("done")
}

func slowRunTest(cm *ContainerManager) {
	log.Println("starting slow run test...")
	for i := 0; i < nIters; i++ {
		err := cm.DockerRun("hello-world", []string{"echo", "\"hello world!\""}, true)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(3 * time.Second)
	}
	log.Println("done")
}

func pauseUnpauseTest(cm *ContainerManager) {
	log.Println("starting pause/unpause test...")
	cm.DockerMakeReady("pausable-start-timer")
	for i := 0; i < nIters; i++ {
		err := cm.DockerPause("pauseable-start-timer")
		if err != nil {
			log.Fatal(err)
		}
		err = cm.DockerUnpause("pausable-start-timer")
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(3 * time.Second)
	}
	log.Println("done")
}
