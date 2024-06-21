package shikari

type ShikariCluster struct {
	Name       string
	NumServers uint8
	NumClients uint8
	Template   string
	EnvVars    []string
	ImgPath    string
}
