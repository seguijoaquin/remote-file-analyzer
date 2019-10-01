package dto

// LauncherDTO holds the info of
type LauncherDTO struct {
	Status bool   `json:"status"`
	Host   string `json:"host"`
	Path   string `json:"path"`
	Data   string `json:"data"`
}
