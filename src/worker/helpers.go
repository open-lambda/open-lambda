package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	dutil "github.com/open-lambda/open-lambda/ol/worker/sandbox/dockerutil"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/embedded"
)

func initOLBaseDir(baseDir string, dockerBaseImage string) error {
	if dockerBaseImage == "" {
		dockerBaseImage = "ol-wasm"
	}

	fmt.Printf("\tExtract '%s' Docker image to %s (make take several minutes).\n", dockerBaseImage, baseDir)

	// PART 1: dump Docker image
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}

	if err = dutil.DumpDockerImage(dockerClient, dockerBaseImage, baseDir); err != nil {
		return err
	}

	// PART 2: various files/dirs on top of the extracted image
	fmt.Printf("\tCreate handler/host/packages/resolve.conf over base image.\n")
	if err := os.Mkdir(path.Join(baseDir, "handler"), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(baseDir, "host"), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(baseDir, "packages"), 0700); err != nil {
		return err
	}

	// need this because Docker containers don't have a dns server in /etc/resolv.conf
	// TODO: make it a config option
	dnsPath := filepath.Join(baseDir, "etc", "resolv.conf")
	if err := ioutil.WriteFile(dnsPath, []byte("nameserver 8.8.8.8\n"), 0644); err != nil {
		return err
	}

	// PART 3: make /dev/* devices
	fmt.Printf("\tCreate /dev/(null,random,urandom) over base image.\n")
	path := filepath.Join(baseDir, "dev", "null")
	if err := exec.Command("mknod", "-m", "0644", path, "c", "1", "3").Run(); err != nil {
		return err
	}

	path = filepath.Join(baseDir, "dev", "random")
	if err := exec.Command("mknod", "-m", "0644", path, "c", "1", "8").Run(); err != nil {
		return err
	}

	path = filepath.Join(baseDir, "dev", "urandom")

	return exec.Command("mknod", "-m", "0644", path, "c", "1", "9").Run()
}

// initOLDir prepares a directory at olPath with necessary files for a
// worker.  This includes default configs and a base directory that is
// used as the root for every lambda instance.
//
// dockerBaseImage specifies what image to extract to the directory
// used as the root FS for lambdas.
//
// Init can be called on a previously initialized directory, even if a
// worker is currently running.  Any worker running will be stopped,
// prior contents deleted, files re-created.  The base dir is a
// special case since it takes so long to populate -- that will be
// reused if it exists (unless newBase is true).
func initOLDir(olPath string, dockerBaseImage string, newBase bool) (err error) {
	initTimePath := filepath.Join(olPath, "ol.init")
	baseDir := common.Conf.SOCK_base_path

	// does the olPath dir already exist?
	if _, err := os.Stat(olPath); !os.IsNotExist(err) {
		// does it contain a previous OL deployment?
		if _, err := os.Stat(initTimePath); !os.IsNotExist(err) {
			fmt.Printf("Previous deployment found at %s.\n", olPath)

			// kill previous worker (if running)
			if err := stopOL(olPath); err != nil {
				return err
			}

			// remove directory contents
			items, err := ioutil.ReadDir(olPath)
			if err != nil {
				return err
			}
			if len(items) > 0 {
				fmt.Printf("Clean previous files in %s\n", olPath)
			}
			for _, item := range items {
				path := filepath.Join(olPath, item.Name())
				if path == baseDir && !newBase {
					fmt.Printf("\tKeep %s\n", path)
					continue
				}
				fmt.Printf("\tRemove %s\n", path)
				if err := os.RemoveAll(path); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("Directory %s already exists but does not contain a previous OL deployment", olPath)
		}
	} else {
		if err := os.Mkdir(olPath, 0700); err != nil {
			return err
		}
	}

	fmt.Printf("Init OL directory at %s\n", olPath)

	if err := ioutil.WriteFile(initTimePath, []byte(time.Now().Local().String()+"\n"), 0400); err != nil {
		return err
	}

	zygoteTreePath := filepath.Join(olPath, "default-zygotes-40.json")
	if err := ioutil.WriteFile(zygoteTreePath, []byte(embedded.DefaultZygotes40_json), 0400); err != nil {
		return err
	}

	confPath := filepath.Join(olPath, "config.json")
	if err := common.SaveConf(confPath); err != nil {
		return err
	}

	if err := os.Mkdir(common.Conf.Worker_dir, 0700); err != nil {
		return err
	}

	if err := os.Mkdir(common.Conf.Registry, 0700); err != nil {
		return err
	}

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		if err := initOLBaseDir(baseDir, dockerBaseImage); err != nil {
			os.RemoveAll(baseDir)
			return err
		}
	} else {
		fmt.Printf("\tReusing prior base at %s (pass -b to reconstruct this)\n", baseDir)
	}

	return nil
}

// Define a custom type for the state
type OlState int

// Define constants for the different states
const (
	Uninitialized OlState = iota
	Running
	StoppedClean
	StoppedDirty
	Unknown = -1
)

// Check the current state of Open Lambda
//
// This function returns the current state of Open Lambda, the PID if possible,
// and an error if it encounters any.
func checkState(olPath string) (OlState, int, error) {
	dirStat, err := os.Stat(olPath)
	if os.IsNotExist(err) {
		// If OL Path doesn't exist, Open Lambda is not initialized.
		return Uninitialized, -1, nil
	}
	if !dirStat.IsDir() {
		return Unknown, -1, fmt.Errorf("olPath is not a directory")
	}

	// Locate the worker.pid file, use it to get the worker's PID
	pidPath := filepath.Join(olPath, "worker.pid")

	data, err := os.ReadFile(pidPath)
	if os.IsNotExist(err) {
		// If we can't find the PID file, it probably means no OL instance is running.
		return StoppedClean, -1, nil
	} else if err != nil {
		// We will be in an unknown state if we encounter any other error.
		return Unknown, -1, fmt.Errorf("unexpected error occurred when reading PID file (%s)", err)
	}

	pidStr := string(data)
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// We will be in an unknown state if the PID file contains any string that is not a number.
		return Unknown, -1, fmt.Errorf("unexpected error occurred when parsing PID file (%s) (%s)", pidStr, err)
	}

	// On Unix systems, FindProcess always succeeds and returns a Process for the given PID,
	// regardless of whether the process exists.
	// https://pkg.go.dev/os#FindProcess
	p, _ := os.FindProcess(pid)
	if err := p.Signal(syscall.Signal(0)); err != nil {
		// If we can't signal the process, it means the process isn't running and yet we found the PID file.
		// Therefore, it was not cleanly shut down.
		return StoppedDirty, -1, nil
	}

	// If we can signal the process, it means the process is currently running.
	return Running, pid, nil
}

// The cleanup procedure for a process that is currently running.
// It should trigger the normal cleanup process of the existing Open Lambda instance.
func gracefulCleanup(p *os.Process) error {
	fmt.Println("Attempting to gracefully shut down the worker process by sending SIGINT.")

	if err := p.Signal(syscall.SIGINT); err != nil {
		fmt.Printf("Failed to send SIGINT to PID %d: %s. Manual cleanup may be required.\n", p.Pid, err.Error())
		return fmt.Errorf("failed to send SIGINT to PID %d: %s", p.Pid, err.Error())
	}

	// Check the process status every 100 milliseconds for up to 60 seconds
	for i := 0; i < 600; i++ {
		err := p.Signal(syscall.Signal(0))
		if err != nil {
			fmt.Println("Worker process stopped successfully.")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("worker process did not stop within 60 seconds")
}

// This function attempts to clean up resources after detecting a dirty shutdown.
// It cleans up cgroups and mounts associated with the Open Lambda instance at `olPath`.
// Returns errors encountered during cleanup operations.
func dirtyCleanup(olPath string) error {
	// Clean up cgroups associated with sandboxes
	cgRoot := filepath.Join("/sys", "fs", "cgroup", filepath.Base(olPath)+"-sandboxes")
	fmt.Printf("Attempting to clean up cgroups at %s\n", cgRoot)

	if files, err := os.ReadDir(cgRoot); err != nil {
		// Log an error if the cgroup root directory cannot be found.
		fmt.Printf("Could not find cgroup root: %s\n", err.Error())
	} else {
		kill := filepath.Join(cgRoot, "cgroup.kill")
		if err := os.WriteFile(kill, []byte(fmt.Sprintf("%d", 1)), os.ModeAppend); err != nil {
			// Log an error if killing processes in the cgroup fails.
			fmt.Printf("Could not kill processes in cgroup: %s\n", err.Error())
		}
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "cg-") {
				cg := filepath.Join(cgRoot, file.Name())
				fmt.Printf("Attempting to remove %s\n", cg)
				if err := syscall.Rmdir(cg); err != nil {
					// Return an error if removing a cgroup fails.
					return fmt.Errorf("could not remove cgroup: %s", err.Error())
				}
			}
		}
		if err := syscall.Rmdir(cgRoot); err != nil {
			// Log an error if removing the cgroup root directory fails.
			fmt.Errorf("could not remove cgroup root: %s", err.Error())
		}
	}

	// Clean up mounts associated with sandboxes
	dirName := filepath.Join(olPath, "worker", "root-sandboxes")
	fmt.Printf("Attempting to clean up mounts at %s\n", dirName)

	if files, err := os.ReadDir(dirName); err != nil {
		// Return an error if the mount root directory cannot be found.
		return fmt.Errorf("could not find mount root: %s", err.Error())
	} else {
		for _, file := range files {
			path := filepath.Join(dirName, file.Name())
			fmt.Printf("Attempting to unmount %s\n", path)
			if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
				// Return an error if unmounting fails.
				return fmt.Errorf("could not unmount: %s", err.Error())
			}
			if err := syscall.Rmdir(path); err != nil {
				// Return an error if removing the mount directory fails.
				return fmt.Errorf("could not remove mount dir: %s", err.Error())
			}
		}
	}

	// Attempt to unmount the main mount directory
	if err := syscall.Unmount(dirName, syscall.MNT_DETACH); err != nil {
		// Log an error if unmounting the main directory fails.
		fmt.Printf("Could not unmount %s: %s\n", dirName, err.Error())
	}

	// Remove the worker.pid file
	if err := os.Remove(filepath.Join(olPath, "worker", "worker.pid")); err != nil {
		// Return an error if removing worker.pid fails.
		return fmt.Errorf("could not remove worker.pid: %s", err.Error())
	}

	return nil
}

// generalCleanup checks the state of Open Lambda and performs appropriate cleanup actions.
// It calls gracefulCleanup for running instances and dirtyCleanup for instances in DirtyShutdown state.
func generalCleanup(olPath string) error {
	state, pid, err := checkState(olPath)
	if err != nil {
		return fmt.Errorf("failed to check OL state: %s", err)
	}

	// Retrieve the process handle for the identified PID.
	p, _ := os.FindProcess(pid)

	switch state {
	case Running:
		err := gracefulCleanup(p)
		if err != nil {
			return fmt.Errorf("failed to gracefully cleanup OL: %s", err)
		}
	case StoppedDirty:
		dirtyCleanup(olPath)
	case Unknown:
		return fmt.Errorf("unknown state detected")
	}

	return nil
}

// stopOL attempts to gracefully stop Open Lambda (OL) and handles errors.
func stopOL(olPath string) error {
	err := generalCleanup(olPath)
	if err != nil {
		fmt.Printf("Error during the shutdown process: %s\n", err)
		fmt.Println("Manual cleanup may be required.")
	}
	return err
}

// cleanupOL performs manual cleanup of Open Lambda (OL) and handles errors.
func cleanupOL(olPath string) error {
	err := generalCleanup(olPath)
	if err != nil {
		fmt.Printf("Error during the manual cleanup process: %s\n", err)
		fmt.Println("You may try to re-run the command.")
	}
	return err
}

// modify the config.json file based on settings from cmdline: -o opt1=val1,opt2=val2,...
//
// apply changes in optsStr to config from confPath, saving result to overridePath
func overrideOpts(confPath, overridePath, optsStr string) error {
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		return err
	}
	conf := make(map[string]any)
	if err := json.Unmarshal(b, &conf); err != nil {
		return err
	}

	opts := strings.Split(optsStr, ",")
	for _, opt := range opts {
		parts := strings.Split(opt, "=")
		if len(parts) != 2 {
			return fmt.Errorf("Could not parse key=val: '%s'", opt)
		}
		keys := strings.Split(parts[0], ".")
		val := parts[1]

		c := conf
		for i := 0; i < len(keys)-1; i++ {
			sub, ok := c[keys[i]]
			if !ok {
				return fmt.Errorf("key '%s' not found", keys[i])
			}
			switch v := sub.(type) {
			case map[string]any:
				c = v
			default:
				return fmt.Errorf("%s refers to a %T, not a map", keys[i], c[keys[i]])
			}
		}

		key := keys[len(keys)-1]
		prev, ok := c[key]
		if !ok {
			return fmt.Errorf("invalid option: '%s'", key)
		}
		switch prev.(type) {
		case string:
			c[key] = val
		case float64:
			c[key], err = strconv.Atoi(val)
			if err != nil {
				return err
			}
		case bool:
			if strings.ToLower(val) == "true" {
				c[key] = true
			} else if strings.ToLower(val) == "false" {
				c[key] = false
			} else {
				return fmt.Errorf("'%s' for %s not a valid boolean value", val, key)
			}
		default:
			return fmt.Errorf("config values of type %T (%s) must be edited manually in the config file ", prev, key)
		}
	}

	// save back config
	s, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(overridePath, s, 0644)
}
