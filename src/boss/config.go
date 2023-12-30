package boss

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
	"github.com/open-lambda/open-lambda/ol/boss/loadbalancer"
)

var Conf *Config

type Config struct {
	Platform   string              `json:"platform"`
	Scaling    string              `json:"scaling"`
	API_key    string              `json:"api_key"`
	Boss_port  string              `json:"boss_port"`
	Worker_Cap int                 `json:"worker_cap"`
	Azure      cloudvm.AzureConfig `json:"azure"`
	Gcp        cloudvm.GcpConfig   `json:"gcp"`
	Lb         string              `json:"lb"`
	MaxGroup   int                 `json:"max_group"`
}

func LoadDefaults() error {
	Conf = &Config{
		Platform:   "mock",
		Scaling:    "manual",
		API_key:    "abc", // TODO
		Boss_port:  "5000",
		Worker_Cap: 20,
		Azure:      *cloudvm.GetAzureConfigDefaults(),
		Gcp:        *cloudvm.GetGcpConfigDefaults(),
		Lb:         "random",
		MaxGroup:   5,
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

	if Conf.Platform == "gcp" {
		cloudvm.LoadGcpConfig(&Conf.Gcp)
	} else if Conf.Platform == "azure" {
		cloudvm.LoadAzureConfig(&Conf.Azure)
	}

	if Conf.Lb == "random" {
		loadbalancer.InitLoadBalancer(loadbalancer.Random, Conf.MaxGroup)
	}
	if Conf.Lb == "sharding" {
		loadbalancer.InitLoadBalancer(loadbalancer.Sharding, Conf.MaxGroup)
	}
	if Conf.Lb == "kmeans" {
		loadbalancer.InitLoadBalancer(loadbalancer.KMeans, Conf.MaxGroup)
	}
	if Conf.Lb == "kmodes" {
		loadbalancer.InitLoadBalancer(loadbalancer.KModes, Conf.MaxGroup)
	}
	if Conf.Lb == "hash" {
		loadbalancer.InitLoadBalancer(loadbalancer.Hash, Conf.MaxGroup)
	}

	return checkConf()
}

func checkConf() error {
	if Conf.Scaling != "manual" && Conf.Scaling != "threshold-scaler" {
		return fmt.Errorf("Scaling type '%s' not implemented", Conf.Scaling)
	}

	return nil
}

// Save writes the Config as an indented JSON to path with 644 mode.
func SaveConf(path string) error {
	s, err := json.MarshalIndent(Conf, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}
