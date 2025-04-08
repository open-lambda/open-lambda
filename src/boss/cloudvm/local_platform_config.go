package cloudvm

import (
	"log"
	"os"
	"path/filepath"
)

type LocalPlatConfig struct {
	Worker_Starting_Port           string `json:"worker_starting_port"`
	Path_To_Worker_Config_Template string `json:"path_to_worker_config_template"`
}

func GetLocalPlatformConfigDefaults() *LocalPlatConfig {
	currPath, err := os.Getwd()
	if err != nil {
		log.Printf("failed to get current path: %v", err)
	}

	return &LocalPlatConfig{
		Worker_Starting_Port:           "6000",
		Path_To_Worker_Config_Template: filepath.Join(currPath, "template.json"),
	}
}
