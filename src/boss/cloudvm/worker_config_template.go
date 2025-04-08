package cloudvm

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/common"
)

func LoadWorkerConfigTemplate(templatePath string, workerPath string, workerPort string) error {
	// Create template.json if it doesn't exist
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// If template.json does not exist, load defaults
		if err := common.LoadDefaults(workerPath); err != nil {
			return fmt.Errorf("failed to load defaults: %v", err)
		}

		// Set the worker port
		common.Conf.Worker_port = GetLocalPlatformConfigDefaults().Worker_Starting_Port

		// Clear the fields that should be patchable later
		common.Conf.Worker_dir = ""
		common.Conf.Registry = ""
		common.Conf.Pkgs_dir = ""
		common.Conf.SOCK_base_path = ""
		common.Conf.Import_cache_tree = ""

		// Save the updated configuration to template.json
		if err := common.SaveConf(templatePath); err != nil {
			return fmt.Errorf("failed to save updated configuration to template.json: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("error checking template.json: %v", err)
	}

	// load template.json
	if err := common.LoadConf(templatePath); err != nil {
		return fmt.Errorf("failed to load template.json: %v", err)
	}

	// Patch fields ONLY if they're empty
	if common.Conf.Worker_dir == "" {
		common.Conf.Worker_dir = filepath.Join(workerPath, "worker")
		log.Printf("Patched Worker_dir: %s", common.Conf.Worker_dir) // TODO: protect it with lock
	}
	if common.Conf.Registry == "" {
		common.Conf.Registry = filepath.Join(workerPath, "registry")
		log.Printf("Patched Registry: %s", common.Conf.Registry) // TODO: protect it with lock
	}
	if common.Conf.Pkgs_dir == "" {
		common.Conf.Pkgs_dir = filepath.Join(workerPath, "lambda", "packages")
		log.Printf("Patched Pkgs_dir: %s", common.Conf.Pkgs_dir) // TODO: protect it with lock
	}
	if common.Conf.SOCK_base_path == "" {
		common.Conf.SOCK_base_path = filepath.Join(workerPath, "lambda")
		log.Printf("Patched SOCK_base_path: %s", common.Conf.SOCK_base_path) // TODO: protect it with lock
	}
	if common.Conf.Import_cache_tree == "" {
		common.Conf.Import_cache_tree = filepath.Join(workerPath, "default-zygotes-40.json")
		log.Printf("Patched Import_cache_tree: %s", common.Conf.Import_cache_tree) // TODO: protect it with lock
	}

	common.Conf.Worker_port = workerPort

	// Save the template configuration to the worker's config directory
	configPath := filepath.Join(workerPath, "config.json")
	if err := common.SaveConf(configPath); err != nil {
		return fmt.Errorf("failed to save updated configuration to worker config: %v", err)
	}

	return nil
}
