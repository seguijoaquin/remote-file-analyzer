package main

// Setup holds the info about the package setup
type Setup struct {
	WorkersDaemon int    `json:"workers_daemon"`
	DaemonURL     string `json:"daemon_url"`
	StatusURL     string `json:"status_url"`
	ProcessorURL  string `json:"processor_url"`
}

// LauncherDTO holds info from launcher params received
type LauncherDTO struct {
	Status bool   `json:"status"`
	Host   string `json:"host"`
	Path   string `json:"path"`
}

// StatusDTO represents the messages the StatusController sends/receives
type StatusDTO struct {
	Host   string `json:"host"`
	Path   string `json:"path"`
	Status string `json:"status"`
	Data   []byte `json:"data"`
}
