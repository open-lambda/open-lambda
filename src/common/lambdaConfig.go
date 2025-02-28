package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var LambdaConf *LambdaConfig

type HTTPTrigger struct {
	Method string `yaml:"method"` // HTTP method (e.g., GET, POST)
}

type CronTrigger struct {
	Schedule string `yaml:"schedule"` // Cron schedule (e.g., "*/5 * * * *")
}

// TODO: add KafkaTrigger struct

// LambdaConfig defines the overall configuration for the lambda function.
type LambdaConfig struct {
	HTTPTriggers []HTTPTrigger `yaml:"http,omitempty"` // List of HTTP triggers
	CronTriggers []CronTrigger `yaml:"cron,omitempty"` // List of cron triggers
	// TODO: add kafka triggers

	// Additional configurations can be added here, such as sandbox settings.
}

// LoadDefaultLambdaConfig initializes the configuration with default values.
func LoadDefaultLambdaConfig() error {
	LambdaConf = &LambdaConfig{
		HTTPTriggers: []HTTPTrigger{
			{
				Method: "POST", // Default HTTP method
			},
		},
	}

	return checkLambdaConfig()
}

// checkLambdaConfig validates the configuration.
func checkLambdaConfig() error {
	if LambdaConf == nil {
		return fmt.Errorf("LambdaConf is not initialized")
	}

	// Validate HTTP triggers
	for _, trigger := range LambdaConf.HTTPTriggers {
		if trigger.Method == "" {
			return fmt.Errorf("HTTP trigger method cannot be empty")
		}
	}

	// Validate cron triggers
	for _, trigger := range LambdaConf.CronTriggers {
		if trigger.Schedule == "" {
			return fmt.Errorf("cron trigger schedule cannot be empty")
		}
	}

	// TODO: validate kafka triggers

	return nil
}

// ParseYaml reads and parses the YAML configuration file.
func ParseYaml(codeDir string) error {
	path := filepath.Join(codeDir, "ol.yaml")
	file, err := os.Open(path)

	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("Config file not found. Loading defaults...")
		return LoadDefaultLambdaConfig()
	} else if err != nil {
		// Failed to open the file
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&LambdaConf) // Use LambdaConf instead of Conf
	if err != nil {
		return fmt.Errorf("failed to parse YAML file: %v", err)
	}

	return checkLambdaConfig()
}
