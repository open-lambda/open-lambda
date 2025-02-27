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
	Path   string `yaml:"path"`   // HTTP endpoint path
	Method string `yaml:"method"` // HTTP method (e.g., GET, POST)
}

type CronTrigger struct {
	Schedule string `yaml:"schedule"` // Cron schedule (e.g., "*/5 * * * *")
}

type KafkaTrigger struct {
	Broker string `yaml:"broker"` // Kafka broker address
	Topic  string `yaml:"topic"`  // Kafka topic
}

// LambdaConfig defines the overall configuration for the lambda function.
type LambdaConfig struct {
	HTTPTriggers  []HTTPTrigger  `yaml:"http,omitempty"`  // List of HTTP triggers
	CronTriggers  []CronTrigger  `yaml:"cron,omitempty"`  // List of cron triggers
	KafkaTriggers []KafkaTrigger `yaml:"kafka,omitempty"` // List of Kafka triggers
	// Additional configurations can be added here, such as sandbox settings.
}

// LoadDefaultLambdaConfig initializes the configuration with default values.
func LoadDefaultLambdaConfig() error {
	LambdaConf = &LambdaConfig{
		HTTPTriggers: []HTTPTrigger{
			{
				Path:   "/",    // Default HTTP endpoint path
				Method: "POST", // Default HTTP method
			},
		},
	}

	return checkLambdaConfig()
}

// checkLambdaConfig validates the configuration.
func checkLambdaConfig() error {
	// Validate HTTP triggers
	for _, trigger := range LambdaConf.HTTPTriggers {
		if trigger.Path == "" {
			return fmt.Errorf("HTTP trigger path cannot be empty")
		}
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

	// Validate Kafka triggers
	for _, trigger := range LambdaConf.KafkaTriggers {
		if trigger.Broker == "" {
			return fmt.Errorf("kafka trigger broker cannot be empty")
		}
		if trigger.Topic == "" {
			return fmt.Errorf("kafka trigger topic cannot be empty")
		}
	}

	return nil
}

// ParseYaml reads and parses the YAML configuration file.
func ParseYaml(codeDir string) error {
	path := filepath.Join(codeDir, "ol.yaml")
	file, err := os.Open(path)

	if errors.Is(err, os.ErrNotExist) {
		// No ol.yaml file found; load the defaults
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
