package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

// Config represents the configuration for a worker server.
type Config struct {
	path     string // where was config file loaded from?
	Registry string `json:"registry"`

	// docker
	Cluster_name  string `json:"cluster_name"`
	Registry_host string `json:"registry_host"`
	Registry_port string `json:"registry_port"`

	// olregistry
	Reg_cluster []string `json:"reg_cluster"`

	// local
	Reg_dir     string `json:"reg_dir"`
	Worker_dir  string `json:"worker_dir"`
	Worker_port string `json:"worker_port"`
	Docker_host string `json:"docker_host"`

	// for unit testing to skip pull path
	Skip_pull_existing bool `json:"Skip_pull_existing"`

	// pass through to sandbox envirenment variable
	Sandbox_config interface{} `json:"sandbox_config"`
}

// SandboxConfJson marshals the Sandbox_config of the Config into a JSON string.
func (c *Config) SandboxConfJson() string {
	s, err := json.Marshal(c.Sandbox_config)
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Dump prints the Config as a JSON string.
func (c *Config) Dump() {
	s, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	log.Printf("CONFIG = %v\n", string(s))
}

// DumpStr returns the Config as an indented JSON string.
func (c *Config) DumpStr() string {
	s, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Save writes the Config as an indented JSON to path with 644 mode.
func (c *Config) Save(path string) error {
	s, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}

// Defaults verifies the fields of Config are correct, and initializes some
// if they are empty.
func (c *Config) Defaults() error {
	if c.Worker_port == "" {
		c.Worker_port = "8080"
	}

	if c.Cluster_name == "" {
		c.Cluster_name = "default"
	}

	if c.Registry == "docker" {
		if c.Registry_host == "" {
			return fmt.Errorf("must specify registry_host\n")
		}

		if c.Registry_port == "" {
			return fmt.Errorf("must specify registry_port\n")
		}
	} else if c.Registry == "olregistry" && len(c.Reg_cluster) == 0 {
		return fmt.Errorf("must specify reg_cluster")
	} else if c.Registry == "local" {
		if c.Reg_dir == "" {
			return fmt.Errorf("must specify local registry directory")
		}

		if !path.IsAbs(c.Reg_dir) {
			if c.path == "" {
				return fmt.Errorf("Reg_dir cannot be relative, unless config is loaded from file")
			}
			path, err := filepath.Abs(path.Join(path.Dir(c.path), c.Reg_dir))
			if err != nil {
				return err
			}
			c.Reg_dir = path
		}
	}

	// worker dir
	if c.Worker_dir == "" {
		return fmt.Errorf("must specify local worker directory")
	}

	if !path.IsAbs(c.Worker_dir) {
		if c.path == "" {
			return fmt.Errorf("Worker_dir cannot be relative, unless config is loaded from file")
		}
		path, err := filepath.Abs(path.Join(path.Dir(c.path), c.Worker_dir))
		if err != nil {
			return err
		}
		c.Worker_dir = path
	}

	// daemon
	if c.Docker_host == "" {
		client, err := docker.NewClientFromEnv()
		if err != nil {
			return fmt.Errorf("failed to get docker client: ", err)
		}

		endpoint := client.Endpoint()
		local := "unix://"
		nonLocal := "https://"
		if strings.HasPrefix(endpoint, local) {
			c.Docker_host = "localhost"
		} else if strings.HasPrefix(endpoint, nonLocal) {
			start := strings.Index(endpoint, nonLocal) + len([]rune(nonLocal))
			end := strings.LastIndex(endpoint, ":")
			c.Docker_host = endpoint[start:end]
		} else {
			return fmt.Errorf("please specify a valid docker host!")
		}
	}

	return nil
}

// ParseConfig reads a file and tries to parse it as a JSON string to a Config
// instance.
func ParseConfig(path string) (*Config, error) {
	config_raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config (%v): %v\n", path, err.Error())
	}
	var config Config

	if err := json.Unmarshal(config_raw, &config); err != nil {
		log.Printf("FILE: %v\n", config_raw)
		return nil, fmt.Errorf("could not parse config (%v): %v\n", path, err.Error())
	}

	config.path = path
	if err := config.Defaults(); err != nil {
		return nil, err
	}

	return &config, nil
}
