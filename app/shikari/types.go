package shikari

type ShikariCluster struct {
	Name       string
	Arch       string
	NumServers uint8
	NumClients uint8
	Template   string
	EnvVars    []string
	ImgPath    string
	Force      bool // flag to whether force operations
}
