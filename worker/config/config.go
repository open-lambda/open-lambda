package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
)

var Timing bool

const REGISTRY_BUCKET = "registry"
const REGISTRY_ACCESS_KEY = "ol_registry_access"
const REGISTRY_SECRET_KEY = "ol_registry_secret"

// Config represents the configuration for a worker server.
type Config struct {
	// base path for path parameters in this config; must be non-empty if any
	// path (e.g., Worker_dir) is relative
	path string
	// registry type: "local" or "olregistry"
	Registry string `json:"registry"`
	// sandbox type: "docker" or "sock"
	// currently ignored as cgroup sandbox is not fully integrated
	Sandbox string `json:"sandbox"`
	// registry directory for storing local copies of handler code
	Registry_dir string `json:"registry_dir"`
	// address of remote registry
	Registry_server string `json:"registry_server"`
	// access key for remote minio registry
	Registry_access_key string `json:"registry_access_key"`
	// secret key for remote minio registry
	Registry_secret_key string `json:"registry_secret_key"`
	// name of the cluster
	Cluster_name string `json:"cluster_name"`
	// pip index address for installing python packages
	Pip_index string `json:"pip_mirror"`
	// directory to install packages to, that sandboxes will read from
	Pkgs_dir string
	// max number of concurrent runners per sandbox
	Max_runners int `json:"max_runners"`

	// cache options
	Handler_cache_size int `json:"handler_cache_size"` //kb
	Import_cache_size  int `json:"import_cache_size"`  //kb

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

	Timing bool `json:"timing"`

	// list of packages to install on startup
	Startup_pkgs []string `json:"startup_pkgs"`

	// which OCI implementation to use for the docker sandbox (e.g., runc or runsc)
	Docker_runtime string `json:"docker_runtime"`
}

// SandboxConfJson marshals the Sandbox_config of the Config into a JSON string.
func (c *Config) SandboxConfJson() string {
	s, err := json.Marshal(c.Sandbox_config)
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Dump prints the Config as a JSON string.
func (c *Config) Dump() {
	s, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	log.Printf("CONFIG = %v\n", string(s))
}

// DumpStr returns the Config as an indented JSON string.
func (c *Config) DumpStr() string {
	s, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Save writes the Config as an indented JSON to path with 644 mode.
func (c *Config) Save(path string) error {
	s, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}

// Defaults verifies the fields of Config are correct, and initializes some
// if they are empty.
func (c *Config) Defaults() error {
	if c.Cluster_name == "" {
		c.Cluster_name = "default"
	}

	if c.Worker_port == "" {
		c.Worker_port = "8080"
	}

	if c.Registry_dir == "" {
		return fmt.Errorf("must specify local registry directory")
	}

	if !path.IsAbs(c.Registry_dir) {
		if c.path == "" {
			return fmt.Errorf("Registry_dir cannot be relative, unless config is loaded from file")
		}
		path, err := filepath.Abs(path.Join(path.Dir(c.path), c.Registry_dir))
		if err != nil {
			return err
		}
		c.Registry_dir = path
	}

	// worker dir
	if c.Worker_dir == "" {
		return fmt.Errorf("must specify local worker directory")
	}

	if !path.IsAbs(c.Worker_dir) {
		if c.path == "" {
			return fmt.Errorf("Worker_dir cannot be relative, unless config is loaded from file")
		}
		path, err := filepath.Abs(path.Join(path.Dir(c.path), c.Worker_dir))
		if err != nil {
			return err
		}
		c.Worker_dir = path
	}

	// sock sandboxes require some extra settings
	if c.Sandbox == "sock" {
		if c.SOCK_base_path == "" {
			return fmt.Errorf("must specify sock_base_path")
		}

		if !path.IsAbs(c.SOCK_base_path) {
			if c.path == "" {
				return fmt.Errorf("sock_base_path cannot be relative unless config is loaded from file")
			}
			path, err := filepath.Abs(path.Join(path.Dir(c.path), c.SOCK_base_path))
			if err != nil {
				return err
			}
			c.SOCK_base_path = path
		}
		c.Pkgs_dir = filepath.Join(c.SOCK_base_path, "packages")
	} else {
		if c.Pkgs_dir == "" {
			return fmt.Errorf("must specify packages directory")
		}

		if !path.IsAbs(c.Pkgs_dir) {
			if c.path == "" {
				return fmt.Errorf("Pkgs_dir cannot be relative, unless config is loaded from file")
			}
			path, err := filepath.Abs(path.Join(path.Dir(c.path), c.Pkgs_dir))
			if err != nil {
				return err
			}
			c.Pkgs_dir = path
		}
	}

	Timing = c.Timing

	return nil
}

// ParseConfig reads a file and tries to parse it as a JSON string to a Config
// instance.
func ParseConfig(path string) (*Config, error) {
	config_raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config (%v): %v\n", path, err.Error())
	}
	var config Config

	if err := json.Unmarshal(config_raw, &config); err != nil {
		log.Printf("FILE: %v\n", config_raw)
		return nil, fmt.Errorf("could not parse config (%v): %v\n", path, err.Error())
	}

	config.path = path
	if err := config.Defaults(); err != nil {
		return nil, err
	}

	return &config, nil
}
