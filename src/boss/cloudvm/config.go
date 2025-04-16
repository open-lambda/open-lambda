package cloudvm

import (
	"encoding/json"
	"log"
)

// TODO: omit. this is already saved in boss config Gcp global variable, no need to have another global variable referring to it
var GcpConf *GcpConfig

type GcpConfig struct {
	DiskSizeGb  int    `json:"disk_size_gb"`
	MachineType string `json:"machine_type"`
}

func GetGcpConfigDefaults() GcpConfig {
	return GcpConfig{
		DiskSizeGb:  30,
		MachineType: "e2-medium",
	}
}

// TODO: omit, no need to load the config to second global variable
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
