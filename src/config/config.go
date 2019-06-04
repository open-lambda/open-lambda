package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
)

var Conf *Config

// Config represents the configuration for a worker server.
type Config struct {
	// location where code packages are stored.  Could be URL or local file path.
	Registry string `json:"registry"`

	// how long should some previously pulled code be used without a check for a newer version?
	Registry_cache_ms int `json:"registry_cache_ms"`

	// sandbox type: "docker" or "sock"
	// currently ignored as cgroup sandbox is not fully integrated
	Sandbox string `json:"sandbox"`

	// name of the cluster
	Cluster_name string `json:"cluster_name"`

	// pip index address for installing python packages
	Pip_index string `json:"pip_mirror"`

	// directory to install packages to, that sandboxes will read from
	Pkgs_dir string

	// max number of concurrent runners per sandbox
	Max_runners int `json:"max_runners"`

	// cache options
	Handler_cache_mb int `json:"handler_cache_mb"`
	Import_cache_mb  int `json:"import_cache_mb"`

	// sandbox options
	// worker directory, which contains handler code, pid file, logs, etc.
	Worker_dir string `json:"worker_dir"`

	// base image path for sock containers
	SOCK_base_path string `json: "sock_base_path"`

	// port the worker server listens to
	Worker_port string `json:"worker_port"`

	// sandbox factory options
	// if sock -> number of cgroup to init
	Cg_pool_size int `json:"cg_pool_size"`

	// for unit testing to skip pull path
	Skip_pull_existing bool `json:"Skip_pull_existing"`

	// pass through to sandbox envirenment variable
	Sandbox_config interface{} `json:"sandbox_config"`

	// write benchmark times to separate log file
	Benchmark_file string `json:"benchmark_log"`

	// should timing info be logged?
	Timing bool `json:"timing"`

	// list of packages to install on startup
	Startup_pkgs []string `json:"startup_pkgs"`

	// which OCI implementation to use for the docker sandbox (e.g., runc or runsc)
	Docker_runtime string `json:"docker_runtime"`
}

// Defaults verifies the fields of Config are correct, and initializes some
// if they are empty.
func LoadDefaults(olPath string) (error) {
	workerDir := filepath.Join(olPath, "worker")
	registryDir := filepath.Join(olPath, "registry")
	baseImgDir := filepath.Join(olPath, "lambda")
	packagesDir := filepath.Join(baseImgDir, "packages")
	
	Conf = &Config{
		Worker_dir:     workerDir,
		Cluster_name:   olPath, // TODO: why?
		Worker_port:    "5000",
		Registry:       registryDir,
		Sandbox:        "sock",
		Pkgs_dir:       packagesDir,
		Sandbox_config: map[string]interface{}{"processes": 10},
		SOCK_base_path: baseImgDir,
		Registry_cache_ms: 5000, // 5 seconds
		Handler_cache_mb: 256, // TODO: base on available mem
		Import_cache_mb: 256, // TODO: base on available mem
	}

	return check()
}

// ParseConfig reads a file and tries to parse it as a JSON string to a Config
// instance.
func LoadFile(path string) (error) {
	config_raw, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not open config (%v): %v\n", path, err.Error())
	}

	if err := json.Unmarshal(config_raw, &Conf); err != nil {
		log.Printf("FILE: %v\n", config_raw)
		return fmt.Errorf("could not parse config (%v): %v\n", path, err.Error())
	}

	return check()
}

func check() error {
	if !path.IsAbs(Conf.Worker_dir) {
		return fmt.Errorf("Worker_dir cannot be relative")
	}

	if Conf.Sandbox == "sock" {
		if Conf.SOCK_base_path == "" {
			return fmt.Errorf("must specify sock_base_path")
		}

		if !path.IsAbs(Conf.SOCK_base_path) {
			return fmt.Errorf("sock_base_path cannot be relative")
		}
	} else {
		if Conf.Pkgs_dir == "" {
			return fmt.Errorf("must specify packages directory")
		}

		if !path.IsAbs(Conf.Pkgs_dir) {
			return fmt.Errorf("Pkgs_dir cannot be relative")
		}
	}

	return nil
}
// SandboxConfJson marshals the Sandbox_config of the Config into a JSON string.
func SandboxConfJson() string {
	s, err := json.Marshal(Conf.Sandbox_config)
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Dump prints the Config as a JSON string.
func Dump() {
	s, err := json.Marshal(Conf)
	if err != nil {
		panic(err)
	}
	log.Printf("CONFIG = %v\n", string(s))
}

// DumpStr returns the Config as an indented JSON string.
func DumpStr() string {
	s, err := json.MarshalIndent(Conf, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Save writes the Config as an indented JSON to path with 644 mode.
func Save(path string) error {
	s, err := json.MarshalIndent(Conf, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}
