package dto

// StatusDTO represents the messages the StatusController sends/receives
type StatusDTO struct {
	Host   string `json:"host"`
	Path   string `json:"path"`
	Status string `json:"status"`
	Data   []byte `json:"data"`
}
