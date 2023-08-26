package cloudvm

import (
	"encoding/json"
	"log"
)

var GcpConf *GcpConfig

type GcpConfig struct {
	DiskSizeGb    int    `json:"disk_size_gb"`
	MachineType   string `json:"machine_type"`
}

func GetGcpConfigDefaults() *GcpConfig {
	return &GcpConfig{
		DiskSizeGb: 30,
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