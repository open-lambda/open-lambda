package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"syscall"
)

var Conf *Config

// Config represents the configuration for a worker server.
type Config struct {
	// worker directory, which contains handler code, pid file, logs, etc.
	Worker_dir string `json:"worker_dir"`

	// port the worker server listens to
	Worker_port string `json:"worker_port"`

	// sandbox type: "docker" or "sock"
	// currently ignored as cgroup sandbox is not fully integrated
	Sandbox string `json:"sandbox"`

	// what kind of server should be launched?  (e.g., lambda or sock)
	Server_mode string `json:"server_mode"`

	// location where code packages are stored.  Could be URL or local file path.
	Registry string `json:"registry"`

	// how long should some previously pulled code be used without a check for a newer version?
	Registry_cache_ms int `json:"registry_cache_ms"`

	// directory to install packages to, that sandboxes will read from
	Pkgs_dir string

	// pip index address for installing python packages
	Pip_index string `json:"pip_mirror"`

	// CACHE OPTIONS
	Mem_pool_mb int `json:"mem_pool_mb"`

	// can be empty (use root zygote only), a JSON obj (specifying
	// the tree), or a path (to a file specifying the tree)
	Import_cache_tree interface{} `json:"import_cache_tree"`

	// base image path for sock containers
	SOCK_base_path string `json: "sock_base_path"`

	// pass through to sandbox envirenment variable
	Sandbox_config interface{} `json:"sandbox_config"`

	// which OCI implementation to use for the docker sandbox (e.g., runc or runsc)
	Docker_runtime string `json:"docker_runtime"`

	Limits   LimitsConfig   `json:"limits"`
	Features FeaturesConfig `json:"features"`
	Trace    TraceConfig    `json:"trace"`
	Storage  StorageConfig  `json:"storage"`
}

type FeaturesConfig struct {
	Reuse_cgroups       bool `json:"reuse_cgroups"`
	Import_cache        bool `json:"import_cache"`
	Downsize_paused_mem bool `json:"downsize_paused_mem"`
}

type TraceConfig struct {
	Cgroups bool `json:"cgroups"`
	Memory  bool `json:"memory"`
	Evictor bool `json:"evictor"`
	Package bool `json:"package"`
}

type StoreString string

func (s StoreString) Mode() StoreMode {
	switch s {
	case "":
		return STORE_REGULAR
	case "memory":
		return STORE_MEMORY
	case "private":
		return STORE_PRIVATE
	default:
		panic(fmt.Errorf("unexpected storage type: '%v'", s))
	}
}

type StorageConfig struct {
	// should be empty, "memory", or "private"
	Root    StoreString `json:"root"`
	Scratch StoreString `json:"scratch"`
	Code    StoreString `json:"code"`
}

type LimitsConfig struct {
	// how many processes can be created within a Sandbox?
	Procs int `json:"procs"`

	// how much memory can a regular lambda use?  The lambda can
	// always set a lower limit for itself.
	Mem_mb int `json:"mem_mb"`

	// how aggresively will the mem of the Sandbox be swapped?
	Swappiness int `json:"swappiness"`

	// how much memory do we use for an admin lambda that is used
	// for pip installs?
	Installer_mem_mb int `json:"installer_mem_mb"`
}

// Defaults verifies the fields of Config are correct, and initializes some
// if they are empty.
func LoadDefaults(olPath string) error {
	workerDir := filepath.Join(olPath, "worker")
	registryDir := filepath.Join(olPath, "registry")
	baseImgDir := filepath.Join(olPath, "lambda")
	packagesDir := filepath.Join(baseImgDir, "packages")

	// split anything above 512 MB evenly between handler and import cache
	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return err
	}
	total_mb := uint64(in.Totalram) * uint64(in.Unit) / 1024 / 1024
	mem_pool_mb := Max(int(total_mb-500), 500)

	Conf = &Config{
		Worker_dir:        workerDir,
		Server_mode:       "lambda",
		Worker_port:       "5000",
		Registry:          registryDir,
		Sandbox:           "sock",
		Pkgs_dir:          packagesDir,
		Sandbox_config:    map[string]interface{}{},
		SOCK_base_path:    baseImgDir,
		Registry_cache_ms: 5000, // 5 seconds
		Mem_pool_mb:       mem_pool_mb,
		Import_cache_tree: "",
		Limits: LimitsConfig{
			Procs:            10,
			Mem_mb:           50,
			Installer_mem_mb: Max(250, Min(500, mem_pool_mb/2)),
			Swappiness:       0,
		},
		Features: FeaturesConfig{
			Import_cache:        true,
			Downsize_paused_mem: true,
		},
		Storage: StorageConfig{
			Root:    "private",
			Scratch: "",
			Code:    "",
		},
	}

	return checkConf()
}

// ParseConfig reads a file and tries to parse it as a JSON string to a Config
// instance.
func LoadConf(path string) error {
	config_raw, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not open config (%v): %v\n", path, err.Error())
	}

	if err := json.Unmarshal(config_raw, &Conf); err != nil {
		log.Printf("FILE: %v\n", config_raw)
		return fmt.Errorf("could not parse config (%v): %v\n", path, err.Error())
	}

	return checkConf()
}

func checkConf() error {
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

		// evictor will ALWAYS try to kill if there's not
		// enough free memory to spin up another container.
		// So we need at least double a memory's needs,
		// otherwise anything running will immediately be
		// evicted.
		//
		// TODO: revise evictor and relax this
		min_mem := 2 * Max(Conf.Limits.Installer_mem_mb, Conf.Limits.Mem_mb)
		if min_mem > Conf.Mem_pool_mb {
			return fmt.Errorf("mem_pool_mb must be at least %d", min_mem)
		}
	} else if Conf.Sandbox == "docker" {
		if Conf.Pkgs_dir == "" {
			return fmt.Errorf("must specify packages directory")
		}

		if !path.IsAbs(Conf.Pkgs_dir) {
			return fmt.Errorf("Pkgs_dir cannot be relative")
		}

		if Conf.Features.Import_cache {
			return fmt.Errorf("features.import_cache must be disabled for docker Sandbox")
		}
	} else {
		return fmt.Errorf("Unknown Sandbox type '%s'", Conf.Sandbox)
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
func DumpConf() {
	s, err := json.Marshal(Conf)
	if err != nil {
		panic(err)
	}
	log.Printf("CONFIG = %v\n", string(s))
}

// DumpStr returns the Config as an indented JSON string.
func DumpConfStr() string {
	s, err := json.MarshalIndent(Conf, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Save writes the Config as an indented JSON to path with 644 mode.
func SaveConf(path string) error {
	s, err := json.MarshalIndent(Conf, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}
