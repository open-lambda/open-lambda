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

// stop OL if running (or just return if not).
// main error scenarios:
// 1. PID exists, but process cannot be killed (worker probably died unexpectedly)
// 2. The cleanup is taking too long (maybe the timeout is insufficient, or there is a deadlock)
func stopOL(_ string) error {
	// locate worker.pid, use it to get worker's PID
	pidPath := filepath.Join(common.Conf.Worker_dir, "worker.pid")
	data, err := ioutil.ReadFile(pidPath)
	if os.IsNotExist(err) {
		fmt.Printf("No worker appears to be running because %s does not exist.\n", pidPath)
		return nil
	} else if err != nil {
		return err
	}
	pidstr := string(data)
	pid, err := strconv.Atoi(pidstr)
	if err != nil {
		return err
	}

	fmt.Printf("According to %s, a worker should already be running (PID %d).\n", pidPath, pid)
	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("Failed to find worker process with PID %d.  May require manual cleanup.\n", pid)
	}
	fmt.Printf("Send SIGINT and wait for worker to exit cleanly.\n")
	if err := p.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("Failed to send SIGINT to PID %d (%s).  May require manual cleanup.\n", pid, err.Error())
	}

	for i := 0; i < 600; i++ {
		err := p.Signal(syscall.Signal(0))
		if err != nil {
			fmt.Printf("OL worker process stopped successfully.\n")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("worker didn't stop after 60s")
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
