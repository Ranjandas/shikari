package lima

type LimaVM struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Memory uint64 `json:"memory"`
	Disk   uint64 `json:"disk"`
	Cpus   int    `json:"cpus"`
}
