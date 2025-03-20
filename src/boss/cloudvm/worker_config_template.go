package cloudvm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/common"
)

func LoadWorkerConfigTemplate(templatePath string, workerPath string) error {
	// Create template.json if it doesn't exist
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// If template.json does not exist, load defaults
		if err := common.LoadDefaults(workerPath); err != nil {
			return fmt.Errorf("failed to load defaults: %v", err)
		}

		// Set the worker port
		common.Conf.Worker_port = GetLocalPlatformConfigDefaults().Worker_Starting_Port

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

	// Save the template configuration to the worker's config directory
	configPath := filepath.Join(workerPath, "config.json")
	if err := common.SaveConf(configPath); err != nil {
		return fmt.Errorf("failed to save updated configuration to worker config: %v", err)
	}

	return nil
}
