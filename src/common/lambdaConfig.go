package common

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

const LambdaConfigFilename = "ol.yaml"

var HandlerNameRegex = regexp.MustCompile(`^[A-Za-z0-9\.\-\_]+$`)

// Triggers defines different ways a lambda can be invoked
type Triggers struct {
	HTTP  []HTTPTrigger  `yaml:"http,omitempty"`  // List of HTTP triggers
	Cron  []CronTrigger  `yaml:"cron,omitempty"`  // List of cron triggers
	Kafka []KafkaTrigger `yaml:"kafka,omitempty"` // List of kafka triggers
}

type HTTPTrigger struct {
	Method string `yaml:"method"` // HTTP method (e.g., GET, POST)
}

type CronTrigger struct {
	Schedule string `yaml:"schedule"` // Cron schedule (e.g., "*/5 * * * *")
}

type KafkaTrigger struct {
	Bootstrap_servers []string `yaml:"bootstrap_servers" json:"bootstrap_servers"` // e.g., ["localhost:9092"]
	Topics            []string `yaml:"topics" json:"topics"`                       // e.g., ["events", "logs"]
	Group_id          string   `yaml:"group_id" json:"group_id"`                   // e.g., "lambda-group"
	Auto_offset_reset string   `yaml:"auto_offset_reset" json:"auto_offset_reset"` // "earliest" or "latest"
}

// LambdaConfig defines the overall configuration for the lambda function.
type LambdaConfig struct {
	Triggers Triggers `yaml:"triggers"` // List of HTTP triggers
	// Additional configurations can be added here.
}

// LoadDefaultLambdaConfig initializes the configuration with default values.
func LoadDefaultLambdaConfig() *LambdaConfig {
	return &LambdaConfig{
		Triggers: Triggers{
			HTTP: []HTTPTrigger{
				{Method: "*"}, // Default to allow all methods
			},
		},
	}
}

// checkLambdaConfig validates the configuration.
func checkLambdaConfig(config *LambdaConfig) error {
	if config == nil {
		return fmt.Errorf("LambdaConfig is not initialized")
	}

	// Validate HTTP triggers
	for _, trigger := range config.Triggers.HTTP {
		if trigger.Method == "" {
			return fmt.Errorf("HTTP trigger method cannot be empty")
		}
	}

	// Validate cron triggers
	for _, trigger := range config.Triggers.Cron {
		if trigger.Schedule == "" {
			return fmt.Errorf("Cron trigger schedule cannot be empty")
		}
	}

	// Validate Kafka triggers
	for _, trigger := range config.Triggers.Kafka {
		if len(trigger.Topics) == 0 {
			return fmt.Errorf("Kafka trigger must have at least one topic")
		}
		if len(trigger.Bootstrap_servers) == 0 {
			return fmt.Errorf("Kafka trigger must specify at least one bootstrap server")
		}
		if trigger.Group_id == "" {
			return fmt.Errorf("Kafka trigger must have a group ID")
		}
	}

	return nil
}

// ParseYaml reads and parses the YAML configuration file.
func LoadLambdaConfig(codeDir string) (*LambdaConfig, error) {
	path := filepath.Join(codeDir, LambdaConfigFilename)
	file, err := os.Open(path)

	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("Config file not found. Loading defaults...")
		return LoadDefaultLambdaConfig(), nil
	} else if err != nil {
		// Failed to open the file
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	var config LambdaConfig

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config) // Use LambdaConf instead of Conf
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML file: %v", err)
	}

	return &config, checkLambdaConfig(&config)
}

func ExtractConfigFromTarGz(tarPath string) (*LambdaConfig, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open lambda tarball: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("invalid .gz file: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid tar: %w", err)
		}

		if header.Name == LambdaConfigFilename {
			var config LambdaConfig
			decoder := yaml.NewDecoder(tr)
			if err := decoder.Decode(&config); err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", LambdaConfigFilename, err)
			}
			return &config, checkLambdaConfig(&config)
		}
	}

	log.Printf("[%s] %s not found, using default config", tarPath, LambdaConfigFilename)
	return LoadDefaultLambdaConfig(), nil
}

// IsHTTPMethodAllowed checks if a method is permitted for this function
func (config *LambdaConfig) IsHTTPMethodAllowed(method string) bool {
	for _, trigger := range config.Triggers.HTTP {
		if trigger.Method == "*" || trigger.Method == method {
			return true
		}
	}
	return false
}

// returns allowed HTTP methods. Used to notify users the allowed https methods when invalid http request was sent.
func (c *LambdaConfig) AllowedHTTPMethods() []string {
	var allowedMethods []string
	for _, trigger := range c.Triggers.HTTP {
		allowedMethods = append(allowedMethods, trigger.Method)
	}
	return allowedMethods
}

func ValidateFunctionName(name string) error {
	if !HandlerNameRegex.MatchString(name) {
		return fmt.Errorf(`invalid function name %q; must match %s`, name, HandlerNameRegex.String())
	}
	return nil
}
