package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var BossConf *Config

type Config struct {
	Platform          string          `json:"platform"`
	Scaling           string          `json:"scaling"`
	API_key           string          `json:"api_key"`
	Boss_port         string          `json:"boss_port"`
	Worker_Cap        int             `json:"worker_cap"`
	Gcp               GcpConfig       `json:"gcp"`
	Local             LocalPlatConfig `json:"local"`
	Lambda_Store_Path string          `json:"lambda_store_path"`
}

func LoadDefaults() error {
	currPath, err := os.Getwd()
	if err != nil {
		log.Printf("failed to get current path: %v", err)
		return err
	}

	BossConf = &Config{
		Platform:          "local",
		Scaling:           "manual",
		API_key:           "abc", // TODO: autogenerate a random key
		Boss_port:         "5000",
		Worker_Cap:        4,
		Gcp:               GetGcpConfigDefaults(),
		Local:             GetLocalPlatformConfigDefaults(),
		Lambda_Store_Path: filepath.Join(currPath, "lambdaStore"),
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

	if err := json.Unmarshal(config_raw, &BossConf); err != nil {
		log.Printf("FILE: %v\n", config_raw)
		return fmt.Errorf("could not parse config (%v): %v\n", path, err.Error())
	}

	return checkConf()
}

func checkConf() error {
	if BossConf.Scaling != "manual" && BossConf.Scaling != "threshold-scaler" {
		return fmt.Errorf("Scaling type '%s' not implemented", BossConf.Scaling)
	}

	return nil
}

// Dump prints the Config as a JSON string.
func DumpConf() {
	s, err := json.Marshal(BossConf)
	if err != nil {
		panic(err)
	}
	log.Printf("CONFIG = %v\n", string(s))
}

// DumpStr returns the Config as an indented JSON string.
func DumpConfStr() string {
	s, err := json.MarshalIndent(BossConf, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Save writes the Config as an indented JSON to path with 644 mode.
func SaveConf(path string) error {
	s, err := json.MarshalIndent(BossConf, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}
