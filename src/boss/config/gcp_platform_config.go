package config

type GcpConfig struct {
	DiskSizeGb     int    `json:"disk_size_gb"`
	MachineType    string `json:"machine_type"`
	LambdaStoreGCS string `json:"lambda_store_gcs"`
}

func GetGcpConfigDefaults() GcpConfig {
	return GcpConfig{
		DiskSizeGb:     30,
		MachineType:    "e2-medium",
		LambdaStoreGCS: "gs://your-bucket-name/lambdas",
	}
}
