package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"os"
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

	Docker          DockerConfig   `json:"docker"`
	Limits          LimitsConfig   `json:"limits"`
	InstallerLimits LimitsConfig   `json:"installer_limits"` // limits profile for installers
	Features        FeaturesConfig `json:"features"`
	Trace           TraceConfig    `json:"trace"`
	Storage         StorageConfig  `json:"storage"`
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

// One unified limits struct for both worker defaults and per-lambda overrides.
// For per-lambda ol.yaml, zero values mean "use worker defaults".
type LimitsConfig struct {
	// process & memory / CPU controls
	Procs       int `json:"procs" yaml:"procs"`
	Mem_mb      int `json:"mem_mb" yaml:"mem_mb"`
	CPU_percent int `json:"cpu_percent" yaml:"cpu_percent"`
	Swappiness  int `json:"swappiness" yaml:"swappiness"`

	// worker default for runtime cap (seconds). Per-lambda may set runtime_sec.
	Max_runtime_default int `json:"max_runtime_default" yaml:"max_runtime_default"`

	// per-lambda override for runtime (seconds). 0 => use Max_runtime_default.
	Runtime_sec int `json:"runtime_sec" yaml:"runtime_sec"`
}

// FillDefaults copies zero fields from def.
func (lc *LimitsConfig) FillDefaults(def LimitsConfig) {
	if lc.Procs == 0 {
		lc.Procs = def.Procs
	}
	if lc.Mem_mb == 0 {
		lc.Mem_mb = def.Mem_mb
	}
	if lc.CPU_percent == 0 {
		lc.CPU_percent = def.CPU_percent
	}
	if lc.Swappiness == 0 {
		lc.Swappiness = def.Swappiness
	}
	if lc.Max_runtime_default == 0 {
		lc.Max_runtime_default = def.Max_runtime_default
	}
	if lc.Runtime_sec == 0 {
		lc.Runtime_sec = def.Runtime_sec
	}
}

// Choose reasonable defaults for a worker deployment (based on memory capacity).
// olPath need not exist (it is used to determine default paths for registry, etc).
func LoadDefaults(olPath string) error {
	cfg, err := GetDefaultWorkerConfig(olPath)
	if err != nil {
		return err
	}

	if err := checkConf(cfg); err != nil {
		return err
	}

	Conf = cfg
	return nil
}

// GetDefaultWorkerConfig returns a config populated with reasonable defaults.
func GetDefaultWorkerConfig(olPath string) (*Config, error) {
	// Check if template.json exists - if so, use it and patch empty fields
	currPath, err := os.Getwd()
	if err == nil {
		// First check current directory
		templatePath := filepath.Join(currPath, "template.json")
		if _, err := os.Stat(templatePath); err != nil {
			// If not found, check parent directory (for workers running in subdirs)
			parentPath := filepath.Dir(currPath)
			templatePath = filepath.Join(parentPath, "template.json")
		}

		if _, err := os.Stat(templatePath); err == nil {
			slog.Info("Loading config from template.json", "path", templatePath)
			cfg, err := ReadInConfig(templatePath)
			if err == nil {
				// Patch worker-specific fields if they're empty (same logic as worker_config_template.go)
				defaultCfg, err := getDefaultConfigForPatching(olPath)
				if err != nil {
					return nil, fmt.Errorf("failed to get defaults for patching: %w", err)
				}

				if cfg.Worker_dir == "" {
					cfg.Worker_dir = defaultCfg.Worker_dir
					slog.Info("Patched Worker_dir", "Worker_dir", cfg.Worker_dir)
				}
				if cfg.Pkgs_dir == "" {
					cfg.Pkgs_dir = defaultCfg.Pkgs_dir
					slog.Info("Patched Pkgs_dir", "Pkgs_dir", cfg.Pkgs_dir)
				}
				if cfg.SOCK_base_path == "" {
					cfg.SOCK_base_path = defaultCfg.SOCK_base_path
					slog.Info("Patched SOCK_base_path", "SOCK_base_path", cfg.SOCK_base_path)
				}
				if cfg.Import_cache_tree == "" {
					cfg.Import_cache_tree = defaultCfg.Import_cache_tree
					slog.Info("Patched Import_cache_tree", "Import_cache_tree", cfg.Import_cache_tree)
				}
				if cfg.Mem_pool_mb == 0 {
					cfg.Mem_pool_mb = defaultCfg.Mem_pool_mb
					slog.Info("Patched Mem_pool_mb", "Mem_pool_mb", cfg.Mem_pool_mb)
				}
				// If template omitted limits, inherit dynamic defaults
				if cfg.Limits == (LimitsConfig{}) {
					cfg.Limits = defaultCfg.Limits
					slog.Info("Patched Limits to defaults")
				}
				if cfg.InstallerLimits == (LimitsConfig{}) {
					cfg.InstallerLimits = defaultCfg.InstallerLimits
					slog.Info("Patched InstallerLimits to defaults")
				}
				// NEW: enforce min required mem pool based on (possibly patched) limits
				minRequired := 2 * Max(cfg.InstallerLimits.Mem_mb, cfg.Limits.Mem_mb)
				if cfg.Mem_pool_mb < minRequired {
					slog.Info("Bumping Mem_pool_mb to satisfy minimum",
						"from", cfg.Mem_pool_mb, "to", minRequired)
					cfg.Mem_pool_mb = minRequired
				}

				return cfg, nil
			}
		}
	}

	// Fallback: generate defaults if no template.json
	return getDefaultConfigForPatching(olPath)
}

// getDefaultConfigForPatching generates the default config used for patching empty template fields
func getDefaultConfigForPatching(olPath string) (*Config, error) {
	var workerDir, registryDir, baseImgDir, zygoteTreePath, packagesDir string

	if olPath != "" {
		workerDir = filepath.Join(olPath, "worker")
		registryDir = filepath.Join(olPath, "registry")
		baseImgDir = filepath.Join(olPath, "lambda")
		zygoteTreePath = filepath.Join(olPath, "default-zygotes-40.json")
		packagesDir = filepath.Join(baseImgDir, "packages")
	}

	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return nil, err
	}
	totalMb := uint64(in.Totalram) * uint64(in.Unit) / 1024 / 1024
	memPoolMb := Max(int(totalMb-500), 500)

	// Sensible defaults
	userLimits := LimitsConfig{
		Procs:               10,
		Mem_mb:              50,
		CPU_percent:         100,
		Max_runtime_default: 30,
		Swappiness:          0,
		// Runtime_sec left 0 => uses Max_runtime_default by default
	}
	// Installers often need more resources; separate profile without the old hack field.
	installerLimits := LimitsConfig{
		Procs:               10,
		Mem_mb:              Max(250, Min(500, memPoolMb/2)),
		CPU_percent:         100,
		Max_runtime_default: 300, // generous default for installs
		Swappiness:          0,
	}

	cfg := &Config{
		Worker_dir:  workerDir,
		Server_mode: "lambda",
		Worker_url:  "localhost",
		Worker_port: "5000",
		// Registry URL with file:// prefix required by gocloud blob backend abstraction.
		// The gocloud library uses URL schemes to route to appropriate storage drivers:
		// file:// for local filesystem, s3:// for AWS S3, gs:// for Google Cloud Storage.
		// Default to local file registry
		Registry:          "file://" + registryDir,
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
		Limits:          userLimits,
		InstallerLimits: installerLimits,
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

	return cfg, nil
}

// ParseConfig reads a file and tries to parse it as a JSON string to a Config
// instance.
func LoadGlobalConfig(path string) error {
	cfg, err := ReadInConfig(path)
	if err != nil {
		return err
	}

	if err := checkConf(cfg); err != nil {
		return err
	}

	Conf = cfg
	return nil
}

func ReadInConfig(path string) (*Config, error) {
	configRaw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config (%v): %w", path, err)
	}

	var templateConfig Config
	if err := json.Unmarshal(configRaw, &templateConfig); err != nil {
		fmt.Printf("Bad config file (%s):\n%s\n", path, string(configRaw))
		return nil, fmt.Errorf("could not parse config (%v): %w", path, err)
	}

	return &templateConfig, nil
}

func checkConf(cfg *Config) error {
	if !path.IsAbs(cfg.Worker_dir) {
		return fmt.Errorf("Worker_dir cannot be relative")
	}

	if cfg.Sandbox == "sock" {
		if cfg.SOCK_base_path == "" {
			return fmt.Errorf("must specify sock_base_path")
		}

		if !path.IsAbs(cfg.SOCK_base_path) {
			return fmt.Errorf("sock_base_path cannot be relative")
		}

		// evictor will ALWAYS try to kill if there's not
		// enough free memory to spin up another container.
		// So we need at least double a memory's needs,
		// otherwise anything running will immediately be
		// evicted.
		//
		// We check against both the regular user limits and the installer limits.
		minMem := 2 * Max(cfg.InstallerLimits.Mem_mb, cfg.Limits.Mem_mb)
		if minMem > cfg.Mem_pool_mb {
			return fmt.Errorf("memPoolMb must be at least %d", minMem)
		}
	} else if cfg.Sandbox == "docker" {
		if cfg.Pkgs_dir == "" {
			return fmt.Errorf("must specify packages directory")
		}

		if !path.IsAbs(cfg.Pkgs_dir) {
			return fmt.Errorf("Pkgs_dir cannot be relative")
		}

		if cfg.Features.Import_cache != "" {
			return fmt.Errorf("features.import_cache must be disabled for docker Sandbox")
		}
	} else {
		return fmt.Errorf("Unknown Sandbox type '%s'", cfg.Sandbox)
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
	slog.Info(fmt.Sprintf("CONFIG = %v", string(s)))
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
func SaveGlobalConfig(path string) error {
	return SaveConfig(Conf, path)
}

// writeConfigToFile writes config data to a file with proper syncing
func writeConfigToFile(cfg *Config, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Marshal config to JSON
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write and sync to ensure data is written to disk
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

func SaveConfig(cfg *Config, path string) error {
	// Write to temp file in same directory to ensure atomic rename
	tempPath := path + ".tmp"
	if err := writeConfigToFile(cfg, tempPath); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return err
	}

	// Atomic rename - this is the key operation that prevents corruption
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Sync the directory to ensure the rename is persisted
	dirFile, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("failed to open directory for sync: %w", err)
	}
	defer dirFile.Close()
	dirFile.Sync() // Ensure directory entry is synced

	slog.Info("Atomically saved config", "path", path)
	return nil
}

func GetOlPath(ctx *cli.Context) (string, error) {
	olPath := ctx.String("path")
	if olPath == "" {
		olPath = "default-ol"
	}
	return filepath.Abs(olPath)
}
