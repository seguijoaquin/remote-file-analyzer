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

type ftpServerRequestDTO struct {
	Message string `json:"message"`
}

type ftpServerResponseDTO struct {
	Message string `json:"message"`
}

type processorRequestDTO struct {
	Host     string `json:"host"`
	Path     string `json:"path"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type processorResponseDTO struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

type persistorRequestDTO struct {
	Host     string `json:"host"`
	Action   string `json:"action"`
	Path     string `json:"path"`
	FileName string `json:"file_name"`
	FileSize int    `json:"file_size"`
}

type persistorResponseDTO struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
