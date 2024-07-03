package lima

type Config struct {
	Env    map[string]string `json:"env,omitempty"`
	Images []Image           `json:"images,omitempty"`
}

type Image struct {
	Location string `json:"location,omitempty"`
	Arch     string `json:"arch,omitempty"`
}

type LimaVM struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Dir    string `json:"dir"`
	Memory uint64 `json:"memory"`
	Disk   uint64 `json:"disk"`
	Cpus   int    `json:"cpus"`
	Config Config `json:"config,omitempty"`
}

type AddrInfo struct {
	Family string `json:"family"`
	Local  string `json:"local"`
}

type Interface struct {
	IfIndex  int        `json:"ifindex"`
	IfName   string     `json:"ifname"`
	AddrInfo []AddrInfo `json:"addr_info"`
}
