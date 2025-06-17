package config

type GcpConfig struct {
	DiskSizeGb     int            `json:"disk_size_gb"`
	MachineType    string         `json:"machine_type"`
	LambdaStoreGCS GCSStoreConfig `json:"lambda_store_gcs"`
}

type GCSStoreConfig struct {
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix"`
}

func GetGcpConfigDefaults() GcpConfig {
	return GcpConfig{
		DiskSizeGb:  30,
		MachineType: "e2-medium",
	}
}
