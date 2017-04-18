package pip

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

type UnpackMirrorServer struct {
	client       *docker.Client
	pipMirror    string
	unpackMirror string
	installed    map[string]bool
}

func NewUnpackMirrorServer(pipMirror string, unpackMirror string) (*UnpackMirrorServer, error) {
	var client *docker.Client
	if c, err := docker.NewClientFromEnv(); err != nil {
		return nil, err
	} else {
		client = c
	}

	if pipMirror == "" {
		pipMirror = "https://pypi.python.org/simple"
	}

	unpackMirror, err := filepath.Abs(unpackMirror)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(unpackMirror, os.ModeDir); err != nil {
		return nil, err
	}

	// Read available packages from file if exists
	p := filepath.Join(unpackMirror, "installed.txt")
	installed := map[string]bool{}
	if file, err := os.Open(p); err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			installed[scanner.Text()] = true
		}
	}

	manager := &UnpackMirrorServer{
		client:       client,
		pipMirror:    pipMirror,
		unpackMirror: unpackMirror,
		installed:    installed,
	}

	return manager, nil
}

// prepare installs a package in the unpack mirror and archives it.
func (m *UnpackMirrorServer) prepare(taskChan, successChan, failChan chan string, group *sync.WaitGroup, commLog *log.Logger) {
	defer group.Done()

	staticBinds := []string{}
	pipMirrorParts := strings.SplitN(m.pipMirror, "://", 2)
	if pipMirrorParts[0] == "file" {
		mirror := filepath.Dir(pipMirrorParts[1])
		staticBinds = append(staticBinds, fmt.Sprintf("%s:%s:ro", mirror, mirror))
	}

	for {
		pkg, ok := <-taskChan
		if !ok {
			successChan <- ""
			return
		}
		if _, ok := m.installed[pkg]; ok {
			continue
		}

		hostDir := filepath.Join(m.unpackMirror, "packages", pkg)
		if err := os.MkdirAll(hostDir, os.ModeDir); err != nil {
			failChan <- pkg
			commLog.Printf("[%v] error during mkdir: %v\n", pkg, err)
			continue
		}

		// installation directory inside container.
		contDir := filepath.Join("/", "packages", pkg)
		binds := append(staticBinds, fmt.Sprintf("%s:%s", hostDir, contDir))

		container, err := m.client.CreateContainer(
			docker.CreateContainerOptions{
				Config: &docker.Config{
					Image: dockerutil.INSTALLER_IMAGE,
					Cmd: []string{
						"pip", "install",
						"-t", contDir,
						"-qqq",
						"-i", m.pipMirror,
						"--no-deps",
						pkg,
					},
				},
				HostConfig: &docker.HostConfig{
					Tmpfs: map[string]string{"tmp": ""},
					Binds: binds,
				},
			},
		)
		if err != nil {
			os.RemoveAll(hostDir)
			commLog.Printf("[%v] fail to create installation container: %v\n", pkg, err)
			failChan <- pkg
			continue
		}

		if err = m.client.StartContainer(container.ID, nil); err != nil {
			os.RemoveAll(hostDir)
			commLog.Printf("[%v] fail to install package: %v\n", pkg, err)
			m.client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
			failChan <- pkg
			continue
		}

		timeout := make(chan bool)
		exitcodeChan := make(chan int)
		errChan := make(chan error)

		go func() {
			time.Sleep(10 * time.Minute)
			timeout <- true
		}()

		go func() {
			if exitcode, err := m.client.WaitContainer(container.ID); err != nil {
				errChan <- err
			} else {
				exitcodeChan <- exitcode
			}
		}()

		select {
		case <-timeout:
			err := m.client.KillContainer(docker.KillContainerOptions{ID: container.ID})
			os.RemoveAll(hostDir)
			if err != nil {
				commLog.Printf("[%v] fail to kill container %s when installation timeout: %v\n", pkg, container.ID, err)
			} else {
				commLog.Printf("[%v] installation timeout\n", pkg)
			}
			m.client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
			failChan <- pkg
			continue
		case err = <-errChan:
			os.RemoveAll(hostDir)
			commLog.Printf("[%v] error during installation: %v\n", pkg, err)
			m.client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
			failChan <- pkg
			continue
		case exitcode := <-exitcodeChan:
			if exitcode != 0 {
				os.RemoveAll(hostDir)
				var buf bytes.Buffer
				err = m.client.Logs(docker.LogsOptions{
					Container:   container.ID,
					ErrorStream: &buf,
					Follow:      true,
					Stderr:      true,
				})
				if err != nil {
					commLog.Printf("[%v] fail to get error logs from installation container: %v\n", pkg, err)
				} else {
					commLog.Printf("[%v] container exited with non-zero code %d: {stderr start}\n%s{stderr end}\n", pkg, exitcode, buf.String())
				}
				m.client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
				failChan <- pkg
				continue
			}
		}

		m.client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})

		// compress files with sources removed
		cmd := exec.Command("tar", "--remove-files", "-zcf", fmt.Sprintf("%s.tar.gz", hostDir), "-C", hostDir, ".")
		if err := cmd.Run(); err != nil {
			var msg string
			if exitErr, ok := err.(*exec.ExitError); ok {
				msg = string(exitErr.Stderr)
			} else {
				msg = err.Error()
			}
			os.RemoveAll(hostDir)
			commLog.Printf("[%v] error when creating package archive: %s\n", pkg, msg)
			failChan <- pkg
			continue
		}

		commLog.Printf("[%v] installation completed\n", pkg)
		successChan <- pkg
	}
}

// Prepare installs a list of package specifications and returns a list of
// remaining ones.
func (m *UnpackMirrorServer) Prepare(pkgs []string) ([]string, error) {
	remains := []string{}

	logPath := filepath.Join(m.unpackMirror, "unpack_mirror.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()
	commLog := log.New(logFile, "", log.LstdFlags)

	installedPath := filepath.Join(m.unpackMirror, "installed.txt")
	installedFile, err := os.OpenFile(installedPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer installedFile.Close()
	installedLog := log.New(installedFile, "", 0)

	group := &sync.WaitGroup{}
	NUM_THREADS := runtime.NumCPU()
	taskChan := make(chan string, NUM_THREADS)
	successChan := make(chan string, NUM_THREADS*2)
	failChan := make(chan string, NUM_THREADS*2)

	group.Add(1)
	go func() {
		defer group.Done()
		count := 0
		for {
			select {
			case success := <-successChan:
				if success == "" {
					count++
					if count == NUM_THREADS {
						return
					}
					continue
				}
				m.installed[success] = true
				installedLog.Printf("%v\n", success)
			case fail := <-failChan:
				remains = append(remains, fail)
			}
		}
	}()

	for i := 0; i < NUM_THREADS; i++ {
		group.Add(1)
		go m.prepare(taskChan, successChan, failChan, group, commLog)
	}

	for idx, pkg := range pkgs {
		commLog.Printf("(%d/%d) Preparing package %v", idx+1, len(pkgs), pkg)
		taskChan <- pkg
	}

	close(taskChan)
	group.Wait()

	return remains, nil
}
