package config

// Setup holds the info about the package setup
type Setup struct {
	WorkersDaemon int    `json:"workers_daemon"`
	DaemonURL     string `json:"daemon_url"`
	StatusURL     string `json:"status_url"`
	ProcessorURL  string `json:"processor_url"`
}
