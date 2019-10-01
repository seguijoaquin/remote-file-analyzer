package main

type launcherRequestDTO struct {
	Report   bool   `json:"report"`
	Host     string `json:"host"`
	Path     string `json:"path"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type statusRequestDTO struct {
	Host    string `json:"host"`
	Path    string `json:"path"`
	Action  string `json:"action"`
	Pending int    `json:"pending"`
}

type statusResponseDTO struct {
	Status  string `json:"status"`
	Pending int    `json:"pending"`
}
