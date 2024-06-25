package lima

type Config struct {
	Env    map[string]string   `json:"env"`
	Images []map[string]string `json:"images"`
}

type LimaVM struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Memory uint64 `json:"memory"`
	Disk   uint64 `json:"disk"`
	Cpus   int    `json:"cpus"`
	Config Config `json:"config"`
}
