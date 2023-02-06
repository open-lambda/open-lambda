package boss

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var Conf *Config

type Config struct {
	Platform  string      `json:"platform"`
	Scaling   string      `json:"scaling"`
	API_key   string      `json:"api_key"`
	Boss_port string      `json:"boss_port"`
	Azure     AzureConfig `json:"azure"`
	Gcp       GcpConfig   `json:"gcp"`
}

type AzureConfig struct {
	Resource_groups rgroups `json:"azure_config"`
}

// Rightnow we default to have only one resource group
type rgroups struct {
	Rgroup    []rgroup `json:"resource_groups"`
	Numrgroup int      `json:"resource_groups_number"`
}

type rgroup struct {
	Resource       armresources.ResourceGroup  `json:"resource_group"`
	Virtual_net    armnetwork.VirtualNetwork   `json:"virtual_network"`
	Subnet         armnetwork.Subnet           `json:"subnet"`
	Public_ip      armnetwork.PublicIPAddress  `json:"public_ip"`
	Security_group armnetwork.SecurityGroup    `json:"security_group"`
	Net_ifc        armnetwork.Interface        `json:"network_interface"`
	Vms            []armcompute.VirtualMachine `json:"virtual_machine"`
	Numvm          int                         `json:"vm_number"`
}

func InitAzureConfig() (*AzureConfig, error) {
	rg := new(rgroup)
	rgs := new(rgroups)
	conf := new(AzureConfig)
	path := "azure.json"
	var content []byte

	rg.Numvm = -1 // this means this rg isn't set up yet
	rgs.Numrgroup = 0
	rgs.Rgroup = append(rgs.Rgroup, *rg)
	conf.Resource_groups = *rgs

	if content, err = json.MarshalIndent(conf, "", "\t"); err != nil {
		return nil, err
	}
	if err = ioutil.WriteFile(path, content, 0666); err != nil {
		return nil, err
	}
	return conf, nil
}

func ReadAzureConfig() (*AzureConfig, error) {
	path := "azure.json"
	_, b := isFile(path)
	var file *os.File
	var err error
	var byteValue []byte

	conf := new(AzureConfig)

	if b {
		if file, err = os.Open(path); err != nil {
			return nil, err
		}
		if byteValue, err = ioutil.ReadAll(file); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(byteValue), conf)
	} else {
		if file, err = os.Create(path); err != nil {
			return nil, err
		}
		if ptr_conf, err := InitAzureConfig(); err != nil {
			return nil, err
		} else {
			conf = ptr_conf
		}
	}

	err = file.Close()
	if err != nil {
		return nil, err
	}
	return conf, err
}

func WriteAzureConfig(conf *AzureConfig) error {
	path := "azure.json"
	var content []byte

	if content, err = json.MarshalIndent(conf, "", "\t"); err != nil {
		return err
	}
	if err = ioutil.WriteFile(path, content, 0666); err != nil {
		return err
	}
	return nil
}

func isExists(path string) (os.FileInfo, bool) {
	f, err := os.Stat(path)
	return f, err == nil || os.IsExist(err)
}

// if its dir
func isDir(path string) (os.FileInfo, bool) {
	f, flag := isExists(path)
	return f, flag && f.IsDir()
}

// if its file
func isFile(path string) (os.FileInfo, bool) {
	f, flag := isExists(path)
	return f, flag && !f.IsDir()
}

type GcpConfig struct {
	// TODO
}

func LoadDefaults() error {
	Conf = &Config{
		Platform:  "mock",
		Scaling:   "manual",
		API_key:   "abc", // TODO
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
