package main

// Setup holds the info about the package setup
type Setup struct {
	WorkersDaemon    int    `json:"workers_daemon"`
	WorkersProcessor int    `json:"workers_processor"`
	DaemonURL        string `json:"daemon_url"`
	StatusURL        string `json:"status_url"`
	ProcessorURL     string `json:"processor_url"`
}

type daemonDTO struct {
	Host     string `json:"host"`
	Path     string `json:"path"`
	User     string `json:"user"`
	Password string `json:"password"`
}
