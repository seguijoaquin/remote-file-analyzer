package main

// Setup holds the info about the package setup
type Setup struct {
	WorkersStatus int    `json:"workers_status"`
	StatusURL     string `json:"status_url"`
	ProcessorURL  string `json:"processor_url"`
}

// StatusDTO represents the messages the StatusController sends/receives
type StatusDTO struct {
	Host   string `json:"host"`
	Path   string `json:"path"`
	Status string `json:"status"`
	Data   []byte `json:"data"`
}
