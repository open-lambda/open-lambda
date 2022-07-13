package boss

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

var Conf *Config

type Config struct {
	Platform string `json:"platform"`
	Scaling string `json:"scaling"`
        API_key string `json:"api_key"`
	Boss_port string  `json:"boss_port"`
	Azure AzureConfig `json:"azure"`
	Gcp GcpConfig `json:"gcp"`
}

type AzureConfig struct {
	// TODO
}

type GcpConfig struct {
	// TODO
}

func LoadDefaults() error {
	Conf = &Config{
		Platform: "mock",
		Scaling: "manual",
		API_key: "abc", // TODO
		Boss_port: "5000",
	}

	return checkConf()
}

// ParseConfig reads a file and tries to parse it as a JSON string to a Config
// instance.
func LoadConf(path string) error {
	config_raw, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not open config (%v): %v\n", path, err.Error())
	}

	if err := json.Unmarshal(config_raw, &Conf); err != nil {
		log.Printf("FILE: %v\n", config_raw)
		return fmt.Errorf("could not parse config (%v): %v\n", path, err.Error())
	}

	return checkConf()
}

func checkConf() error {
	if Conf.Scaling != "manual" {
		return fmt.Errorf("Scaling type '%s' not implemented", Conf.Scaling)
	}

	return nil
}

// Dump prints the Config as a JSON string.
func DumpConf() {
	s, err := json.Marshal(Conf)
	if err != nil {
		panic(err)
	}
	log.Printf("CONFIG = %v\n", string(s))
}

// DumpStr returns the Config as an indented JSON string.
func DumpConfStr() string {
	s, err := json.MarshalIndent(Conf, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Save writes the Config as an indented JSON to path with 644 mode.
func SaveConf(path string) error {
	s, err := json.MarshalIndent(Conf, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}
