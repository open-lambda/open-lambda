package cloudvm

import (
	"encoding/json"
	"log"

	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var GcpConf *GcpConfig
var AzureConf *AzureConfig

type GcpConfig struct {
	DiskSizeGb  int    `json:"disk_size_gb"`
	MachineType string `json:"machine_type"`
}

func GetGcpConfigDefaults() *GcpConfig {
	return &GcpConfig{
		DiskSizeGb:  30,
		MachineType: "e2-medium",
	}
}

func LoadGcpConfig(newConf *GcpConfig) {
	GcpConf = newConf
}

// Dump prints the Config as a JSON string.
func DumpConf() {
	s, err := json.Marshal(GcpConf)
	if err != nil {
		panic(err)
	}
	log.Printf("CONFIG = %v\n", string(s))
}

// DumpStr returns the Config as an indented JSON string.
func DumpConfStr() string {
	s, err := json.MarshalIndent(GcpConf, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

type AzureConfig struct {
	Resource_groups rgroups `json:"azure_config"`
}

// TODO: Rightnow we default to have only one resource group
type rgroups struct {
	Rgroup    []rgroup `json:"resource_groups"`
	Numrgroup int      `json:"resource_groups_number"`
}

type rgroup struct {
	Resource armresources.ResourceGroup `json:"resource_group"`
	Vms      []vmStatus                 `json:"virtual_machine_status"`
	Numvm    int                        `json:"vm_number"`
	SSHKey   string                     `json:"ssh_key"`
}

type vmStatus struct {
	Status         string                     `json:"virtual_machine_status"`
	Vm             armcompute.VirtualMachine  `json:"virtual_machine"`
	Virtual_net    armnetwork.VirtualNetwork  `json:"virtual_network"`
	Subnet         armnetwork.Subnet          `json:"subnet"`
	Public_ip      armnetwork.PublicIPAddress `json:"public_ip"`
	Security_group armnetwork.SecurityGroup   `json:"security_group"`
	Net_ifc        armnetwork.Interface       `json:"network_interface"`
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

func LoadAzureConfig(newConf *AzureConfig) {
	AzureConf = newConf
}

func GetAzureConfigDefaults() *AzureConfig {
	rg := &rgroup{
		Numvm:  0,
		SSHKey: "~/.ssh/ol-boss_key.pem",
	}

	rgs := &rgroups{
		Numrgroup: 1,
	}
	rgs.Rgroup = append(rgs.Rgroup, *rg)

	conf := &AzureConfig{
		Resource_groups: *rgs,
	}

	path := "azure.json"
	var content []byte
	content, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		panic(err)
	}

	if err = ioutil.WriteFile(path, content, 0666); err != nil {
		panic(err)
	}
	return conf
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
		conf = GetAzureConfigDefaults()
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

	content, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(path, content, 0666); err != nil {
		return err
	}
	return nil
}
