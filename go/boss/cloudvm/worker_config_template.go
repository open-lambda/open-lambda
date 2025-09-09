package cloudvm

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/open-lambda/open-lambda/go/boss/config"
	"github.com/open-lambda/open-lambda/go/common"
)

// SaveTemplateConfToWorkerDir constructs a worker-specific config using:
// 1. The global template config (shared defaults)
// 2. Worker-specific defaults based on its directory path
// 3. A unique worker port number
// The final config is then written to <workerPath>/config.json.
func SaveTemplateConfToWorkerDir(cfg *common.Config, workerPath string, workerPort string) error {
	// Copy the config so we can safely mutate it
	cfgCopy := *cfg

	defaultCfg, err := common.GetDefaultWorkerConfig(workerPath)
	if err != nil {
		return fmt.Errorf("failed to get default worker config: %v", err)
	}

	// Patch fields ONLY if they're empty
	if cfgCopy.Worker_dir == "" {
		cfgCopy.Worker_dir = defaultCfg.Worker_dir
		slog.Info(fmt.Sprintf("Patched Worker_dir: %s", cfg.Worker_dir))
	}
	if cfgCopy.Registry == "" {
		cfgCopy.Registry = defaultCfg.Registry
		slog.Info(fmt.Sprintf("Patched Registry: %s", cfg.Registry))
	}
	if cfgCopy.Pkgs_dir == "" {
		cfgCopy.Pkgs_dir = defaultCfg.Pkgs_dir
		slog.Info(fmt.Sprintf("Patched Pkgs_dir: %s", cfg.Pkgs_dir))
	}
	if cfgCopy.SOCK_base_path == "" {
		cfgCopy.SOCK_base_path = defaultCfg.SOCK_base_path
		slog.Info(fmt.Sprintf("Patched SOCK_base_path: %s", cfg.SOCK_base_path))
	}
	if cfgCopy.Import_cache_tree == "" {
		cfgCopy.Import_cache_tree = defaultCfg.Import_cache_tree
		slog.Info(fmt.Sprintf("Patched Import_cache_tree: %s", cfg.Import_cache_tree))
	}

	cfgCopy.Worker_port = workerPort

	// point the worker registry to lambda store (platform-aware)
	cfgCopy.Registry = config.BossConf.GetLambdaStoreURL()

	// Save the template configuration to the worker's config directory
	configPath := filepath.Join(workerPath, "config.json")
	return common.SaveConfig(&cfgCopy, configPath)
}
