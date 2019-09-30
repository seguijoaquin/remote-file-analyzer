package dto

// DaemonDTO holds the info of
type DaemonDTO struct {
	Host     string `json:"host"`
	Path     string `json:"path"`
	User     string `json:"user"`
	Password string `json:"password"`
}
