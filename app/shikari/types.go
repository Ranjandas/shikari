package shikari

type ShikariCluster struct {
	Name       string
	Arch       string
	NumServers uint8
	NumClients uint8
	ServersSet bool // whether the number of servers was given, used by scale
	ClientsSet bool // whether the number of clients was given, used by scale
	Template   string
	EnvVars    []string
	ImgPath    string
	Force      bool // flag to whether force operations
}
