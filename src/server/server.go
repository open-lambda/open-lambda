package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/open-lambda/open-lambda/ol/config"
)

func Main() error {
	var s interface {
		cleanup()
	}

	pidPath := filepath.Join(config.Conf.Worker_dir, "worker.pid")
	if _, err := os.Stat(pidPath); err == nil {
		return fmt.Errorf("previous working may be running, %s already exists", pidPath)
	} else if !os.IsNotExist(err) {
		// we were hoping to get the not-exist error, but got something else unexpected
		return err
	}

	// start with a fresh env
	if err := os.RemoveAll(config.Conf.Worker_dir); err != nil {
		return err
	} else if err := os.MkdirAll(config.Conf.Worker_dir, 0700); err != nil {
		return err
	}

	log.Printf("save PID %d to file %s", os.Getpid(), pidPath)
	if err := ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		return err
	}

	switch config.Conf.Server_mode {
	case "lambda":
		s = LambdaMain()
	case "sock":
		s = SockMain()
	default:
		return fmt.Errorf("unknown Server_mode %s", config.Conf.Server_mode)
	}

	// clean up if signal hits us
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		log.Printf("received kill signal, cleaning up")
		s.cleanup()
		log.Printf("remove worker.pid")
		os.Remove(pidPath)
		log.Printf("exiting")
		os.Exit(1)
	}()

	port := fmt.Sprintf(":%s", config.Conf.Worker_port)
	log.Fatal(http.ListenAndServe(port, nil))
	panic("ListenAndServe should never return")
}
