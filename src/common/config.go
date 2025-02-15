package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"syscall"

	"github.com/urfave/cli/v2"
)

// Configuration is stored globally here
var Conf *Config

// Config represents the configuration for a worker server.
type Config struct {
	// worker directory, which contains handler code, pid file, logs, etc.
	Worker_dir string `json:"worker_dir"`

	// Url/ip the worker server listens to
	Worker_url string `json:"worker_url"`

	// port the worker server listens to
	Worker_port string `json:"worker_port"`

	// log output of the runtime and proxy?
	Log_output bool `json:"log_output"`

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
	Import_cache_tree any `json:"import_cache_tree"`

	// base image path for sock containers
	SOCK_base_path string `json:"sock_base_path"`

	// pass through to sandbox envirenment variable
	Sandbox_config any `json:"sandbox_config"`

	Docker   DockerConfig   `json:"docker"`
	Limits   LimitsConfig   `json:"limits"`
	Features FeaturesConfig `json:"features"`
	Trace    TraceConfig    `json:"trace"`
	Storage  StorageConfig  `json:"storage"`
}

type DockerConfig struct {
	// which OCI implementation to use for the docker sandbox (e.g., runc or runsc)
	Runtime string `json:"runtime"`
	// name of the image used for Docker containers
	Base_image string `json:"base_image"`
}

type FeaturesConfig struct {
	Reuse_cgroups       bool   `json:"reuse_cgroups"`
	Import_cache        string `json:"import_cache"`
	Downsize_paused_mem bool   `json:"downsize_paused_mem"`
	Enable_seccomp      bool   `json:"enable_seccomp"`
}

type TraceConfig struct {
	Cgroups bool `json:"cgroups"`
	Memory  bool `json:"memory"`
	Evictor bool `json:"evictor"`
	Package bool `json:"package"`
	Latency bool `json:"latency"`
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

	// what percent of a core can it use per period?  (0-100, or more for multiple cores)
	CPU_percent int `json:"cpu_percent"`

	// how many seconds can Lambdas run?  (maybe be overridden on per-lambda basis)
	Max_runtime_default int `json:"max_runtime_default"`

	// how aggressively will the mem of the Sandbox be swapped?
	Swappiness int `json:"swappiness"`

	// how much memory do we use for an admin lambda that is used
	// for pip installs?
	Installer_mem_mb int `json:"installer_mem_mb"`
}

// Choose reasonable defaults for a worker deployment (based on memory capacity).
// olPath need not exist (it is used to determine default paths for registry, etc).
func LoadDefaults(olPath string) error {
	workerDir := filepath.Join(olPath, "worker")
	registryDir := filepath.Join(olPath, "registry")
	baseImgDir := filepath.Join(olPath, "lambda")
	zygoteTreePath := filepath.Join(olPath, "default-zygotes-40.json")
	packagesDir := filepath.Join(baseImgDir, "packages")

	// split anything above 512 MB evenly between handler and import cache
	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return err
	}
	totalMb := uint64(in.Totalram) * uint64(in.Unit) / 1024 / 1024
	memPoolMb := Max(int(totalMb-500), 500)

	Conf = &Config{
		Worker_dir:        workerDir,
		Server_mode:       "lambda",
		Worker_url:        "localhost",
		Worker_port:       "5000",
		Registry:          registryDir,
		Sandbox:           "sock",
		Log_output:        true,
		Pkgs_dir:          packagesDir,
		Sandbox_config:    map[string]any{},
		SOCK_base_path:    baseImgDir,
		Registry_cache_ms: 5000, // 5 seconds
		Mem_pool_mb:       memPoolMb,
		Import_cache_tree: zygoteTreePath,
		Docker: DockerConfig{
			Base_image: "ol-min",
		},
		Limits: LimitsConfig{
			Procs:               10,
			Mem_mb:              50,
			CPU_percent:         100,
			Max_runtime_default: 30,
			Installer_mem_mb:    Max(250, Min(500, memPoolMb/2)),
			Swappiness:          0,
		},
		Features: FeaturesConfig{
			Import_cache:        "tree",
			Downsize_paused_mem: true,
			Enable_seccomp:      true,
		},
		Trace: TraceConfig{
			Cgroups: false,
			Memory:  false,
			Evictor: false,
			Package: false,
			Latency: false,
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
	configRaw, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not open config (%v): %v", path, err.Error())
	}

	if err := json.Unmarshal(configRaw, &Conf); err != nil {
		fmt.Printf("Bad config file (%s):\n%s\n", path, string(configRaw))
		return fmt.Errorf("could not parse config (%v): %v", path, err.Error())
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
		minMem := 2 * Max(Conf.Limits.Installer_mem_mb, Conf.Limits.Mem_mb)
		if minMem > Conf.Mem_pool_mb {
			return fmt.Errorf("memPoolMb must be at least %d", minMem)
		}
	} else if Conf.Sandbox == "docker" {
		if Conf.Pkgs_dir == "" {
			return fmt.Errorf("must specify packages directory")
		}

		if !path.IsAbs(Conf.Pkgs_dir) {
			return fmt.Errorf("Pkgs_dir cannot be relative")
		}

		if Conf.Features.Import_cache != "" {
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

func GetOlPath(ctx *cli.Context) (string, error) {
	olPath := ctx.String("path")
	if olPath == "" {
		olPath = "default-ol"
	}
	return filepath.Abs(olPath)
}
