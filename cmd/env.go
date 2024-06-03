/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	lima "github.com/ranjandas/shikari/app"
	"github.com/spf13/cobra"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Prints client config environment variables",
	Long: `Prints client config environment variables.
For example:

export CONSUL_HTTP_TOKEN=xxxxx
export CONSUL_HTTP_ADDR=http://x.x.x.x:xxxx`,
	Run: func(cmd *cobra.Command, args []string) {

		if clientConfigOpts.Name != "" {
			printClientConfigEnvs(clientConfigOpts)
		}

	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// envCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// envCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	envCmd.Flags().StringVarP(&clientConfigOpts.Name, "name", "n", "", "name of the cluster")
	envCmd.Flags().BoolVarP(&clientConfigOpts.ACL, "acl", "", false, "prints the ACL token variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.TLS, "tls", "", false, "prints the TLS variables")
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

type ClientConfigOpts struct {
	Name string
	TLS  bool
	ACL  bool
}

var clientConfigOpts ClientConfigOpts

func printClientConfigEnvs(config ClientConfigOpts) {

	ipAddress := getIPAddress(getRandomServer(config))

	fmt.Println(consulVariables(config, ipAddress))
	fmt.Println(nomadVariables(config, ipAddress))
}

func getRandomServer(config ClientConfigOpts) string {
	instances := lima.GetInstancesByPrefix(config.Name)
	runningInstances := lima.GetInstancesByStatus(instances, "running")

	if !(len(runningInstances) > 0) {
		os.Exit(1)
	}

	randomIndex := rand.Intn(len(runningInstances))
	return runningInstances[randomIndex].Name
}

func getIPAddress(srvName string) string {

	command := "ip -j addr show dev lima0"
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("limactl shell %s %s", srvName, command))

	output, err := cmd.Output()

	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	var interfaces []Interface
	err = json.Unmarshal([]byte(output), &interfaces)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	// Extract the local address where family is "inet"
	for _, iface := range interfaces {
		for _, addrInfo := range iface.AddrInfo {
			if addrInfo.Family == "inet" {
				return addrInfo.Local
			}
		}
	}
	return ""
}

func consulVariables(config ClientConfigOpts, addr string) string {

	scheme := "http://"
	port := 8500
	bootstrapToken := "root" // consul bootstrap token

	if config.TLS {
		scheme = "https://"
		port = 8501
	}

	consulHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	consulHTTPAddrVar := fmt.Sprintf("CONSUL_HTTP_ADDR=%s", consulHTTPAddr)
	consulTokenVar := fmt.Sprintf("CONSUL_HTTP_TOKEN=%s", bootstrapToken)

	if !config.ACL {
		return consulHTTPAddrVar
	}

	return strings.Join([]string{consulHTTPAddrVar, consulTokenVar}, "\n")
}

func nomadVariables(config ClientConfigOpts, addr string) string {

	scheme := "http://"
	port := 4646
	bootstrapToken := "00000000-0000-0000-0000-000000000000" // consul bootstrap token

	if config.TLS {
		scheme = "https://"
	}

	consulHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	consulHTTPAddrVar := fmt.Sprintf("CONSUL_HTTP_ADDR=%s", consulHTTPAddr)
	consulTokenVar := fmt.Sprintf("CONSUL_HTTP_TOKEN=%s", bootstrapToken)

	if !config.ACL {
		return consulHTTPAddrVar
	}

	return strings.Join([]string{consulHTTPAddrVar, consulTokenVar}, "\n")
}
