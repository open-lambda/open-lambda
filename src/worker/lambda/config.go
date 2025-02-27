package lambda

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var Conf *Config

const (
	HTTP_TRIGGER  = "http"
	CRON_TRIGGER  = "cron"
	KAFKA_TRIGGER = "kafka"
)

// Trigger defines the configuration for a function trigger.
type Trigger struct {
	Type     string `yaml:"type"`               // Type of trigger: http, cron, kafka
	Schedule string `yaml:"schedule,omitempty"` // Cron schedule (for CRON_TRIGGER)
	Topic    string `yaml:"topic,omitempty"`    // Kafka topic (for KAFKA_TRIGGER)
	GroupID  string `yaml:"group_id,omitempty"` // Kafka consumer group (for KAFKA_TRIGGER)
}

// Config defines the overall configuration for the lambda function.
type Config struct {
	Triggers []Trigger `yaml:"triggers"`
	// Additional configurations can be added here, such as sandbox settings.
}

// LoadDefaults initializes the configuration with default values.
func LoadDefaults() error {
	Conf = &Config{
		Triggers: []Trigger{
			{
				Type: HTTP_TRIGGER, // Default to HTTP trigger
			},
		},
	}

	return checkConf()
}

// checkConf validates the configuration.
func checkConf() error {
	for _, trigger := range Conf.Triggers {
		switch trigger.Type {
		case HTTP_TRIGGER, CRON_TRIGGER, KAFKA_TRIGGER:
			// Valid trigger type
		default:
			return fmt.Errorf("trigger type is not implemented: %s", trigger.Type)
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
		return LoadDefaults()
	} else if err != nil {
		// Failed to open the file
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&Conf)
	if err != nil {
		return fmt.Errorf("failed to parse YAML file: %v", err)
	}

	return checkConf()
}
