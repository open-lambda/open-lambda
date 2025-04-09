package cloudvm

import (
	"encoding/json"
	"log"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/common"
)

func SaveTemplateConfToWorkerDir(cfg *common.Config, workerPath string, workerPort string) error {
	// Copy the config so we can safely mutate it
	cfgCopy := deepCopyConfig(cfg)
	defaultCfg, _ := common.GetDefaultWorkerConfig(workerPath)

	// Patch fields ONLY if they're empty
	if cfgCopy.Worker_dir == "" {
		cfgCopy.Worker_dir = defaultCfg.Worker_dir
		log.Printf("Patched Worker_dir: %s", cfg.Worker_dir)
	}
	if cfgCopy.Registry == "" {
		cfgCopy.Registry = defaultCfg.Registry
		log.Printf("Patched Registry: %s", cfg.Registry)
	}
	if cfgCopy.Pkgs_dir == "" {
		cfgCopy.Pkgs_dir = defaultCfg.Pkgs_dir
		log.Printf("Patched Pkgs_dir: %s", cfg.Pkgs_dir)
	}
	if cfgCopy.SOCK_base_path == "" {
		cfgCopy.SOCK_base_path = defaultCfg.SOCK_base_path
		log.Printf("Patched SOCK_base_path: %s", cfg.SOCK_base_path)
	}
	if cfgCopy.Import_cache_tree == "" {
		cfgCopy.Import_cache_tree = defaultCfg.Import_cache_tree
		log.Printf("Patched Import_cache_tree: %s", cfg.Import_cache_tree)
	}
	cfgCopy.Worker_port = workerPort

	// Save the template configuration to the worker's config directory
	configPath := filepath.Join(workerPath, "config.json")
	return common.SaveTemplateConf(cfgCopy, configPath)
}

func deepCopyConfig(src *common.Config) *common.Config {
	bytes, _ := json.Marshal(src)
	var dst common.Config
	json.Unmarshal(bytes, &dst)
	return &dst
}
