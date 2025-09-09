package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type LocalPlatConfig struct {
	Worker_Starting_Port           string `json:"worker_starting_port"`
	Path_To_Worker_Config_Template string `json:"path_to_worker_config_template"`
	LambdaStoreLocal               string `json:"lambda_store_local"`
}

func GetLocalPlatformConfigDefaults() LocalPlatConfig {
	currPath, err := os.Getwd()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get current path: %v", err))
	}

	return LocalPlatConfig{
		Worker_Starting_Port:           "6000",
		Path_To_Worker_Config_Template: filepath.Join(currPath, "template.json"),
		LambdaStoreLocal:               "file://" + filepath.Join(currPath, "lambdaStore"),
	}
}
