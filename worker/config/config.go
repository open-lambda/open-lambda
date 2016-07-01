package config

type Config struct {
	Registry_host string `json:"registry_host"`
	Registry_port string `json:"registry_port"`
	Docker_host   string `json:"docker_host"`
	// for unit testing to skip pull path
	Skip_pull_existing bool `json:"Skip_pull_existing"`
}
