package boss

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
	Tree_path  string              `json:"tree_path"`
	Worker_mem int                 `json:"worker_mem"`
}

func LoadDefaults() error {
	olPath, err := os.Getwd()
	if err != nil {
		log.Println("Error getting executable path:", err)
		return err
	}
	tree_path := fmt.Sprintf("%s/default-zygote-40.json", olPath)

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
		Tree_path:  tree_path,
		Worker_mem: 32768,
	}

	return checkConf()
}

func Max(x int, y int) int {
	if x > y {
		return x
	}

	return y
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

	cloudvm.LoadTreePath(Conf.Tree_path)
	cloudvm.LoadWorkerMem(Conf.Worker_mem)
	if Conf.Platform == "gcp" {
		cloudvm.LoadGcpConfig(&Conf.Gcp)
	} else if Conf.Platform == "azure" {
		cloudvm.LoadAzureConfig(&Conf.Azure)
	}

	if Conf.Lb == "random" {
		loadbalancer.InitLoadBalancer(loadbalancer.Random, Conf.MaxGroup, Conf.Tree_path)
	}
	if Conf.Lb == "sharding" {
		loadbalancer.InitLoadBalancer(loadbalancer.Sharding, Conf.MaxGroup, Conf.Tree_path)
	}
	if Conf.Lb == "kmeans" {
		loadbalancer.InitLoadBalancer(loadbalancer.KMeans, Conf.MaxGroup, Conf.Tree_path)
	}
	if Conf.Lb == "kmodes" {
		loadbalancer.InitLoadBalancer(loadbalancer.KModes, Conf.MaxGroup, Conf.Tree_path)
	}
	if Conf.Lb == "hashfunc" {
		loadbalancer.InitLoadBalancer(loadbalancer.HashFunc, Conf.MaxGroup, Conf.Tree_path)
	}
	if Conf.Lb == "hashzygote" {
		loadbalancer.InitLoadBalancer(loadbalancer.HashZygote, Conf.MaxGroup, Conf.Tree_path)
	}

	return checkConf()
}

func checkConf() error {
	if Conf.Scaling != "manual" && Conf.Scaling != "threshold-scaler" {
		return fmt.Errorf("Scaling type '%s' not implemented", Conf.Scaling)
	}
	if Conf.Lb != "random" && Conf.Lb != "sharding" && Conf.Lb != "kmeans" && Conf.Lb != "kmodes" && Conf.Lb != "hashfunc" && Conf.Lb != "hashzygote" {
		return fmt.Errorf("%s is not implemented", Conf.Lb)
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
