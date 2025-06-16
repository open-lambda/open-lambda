package config

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
