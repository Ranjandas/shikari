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

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Prints client config environment variables",
	Long: `Prints client config environment variables.
For example:
shikari env -n murphy -tai consul
export CONSUL_HTTP_ADDR=https://xxx.xxx
export CONSUL_HTTP_TOKEN=xxx
export CONSUL_HTTP_SSL_VERIFY=false`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 && clientConfigOpts.Name != "" {
			clientConfigOpts.printClientConfigEnvs(args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	envCmd.Flags().StringVarP(&clientConfigOpts.Name, "name", "n", "", "name of the cluster")
	envCmd.Flags().BoolVarP(&clientConfigOpts.ACL, "acl", "a", false, "prints the ACL token variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.TLS, "tls", "t", false, "prints the TLS variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.Insecure, "insecure", "i", false, "prints the skip TLS Verify variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.Unset, "unset", "u", false, "unset the variables insetad of export")
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
	Name     string
	TLS      bool
	ACL      bool
	Insecure bool
	Unset    bool
}

var clientConfigOpts ClientConfigOpts

func (c ClientConfigOpts) printClientConfigEnvs(product string) {
	switch product {
	case "consul":
		fmt.Println(c.getConsulVariables())
	case "nomad":
		fmt.Println(c.getNomadVariables())
	case "vault":
		fmt.Println(c.getVaultVariables())
	}
}

func (c ClientConfigOpts) getRandomServer() string {
	// always get the instances of type server "-srv"
	instances := lima.GetInstancesByPrefix(fmt.Sprintf("%s-srv", c.Name))
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

func (c ClientConfigOpts) getConsulVariables() string {

	if c.Unset {
		return "unset CONSUL_HTTP_ADDR\nunset CONSUL_HTTP_TOKEN\nunset CONSUL_HTTP_SSL_VERIFY"
	}

	addr := getIPAddress(c.getRandomServer())
	scheme := "http://"
	port := 8500
	bootstrapToken := "root" // consul bootstrap token

	if c.TLS {
		scheme = "https://"
		port = 8501
	}

	consulHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export CONSUL_HTTP_ADDR=%s", consulHTTPAddr)
	tokenVar := fmt.Sprintf("export CONSUL_HTTP_TOKEN=%s", bootstrapToken)
	insecureTLSVar := "export CONSUL_HTTP_SSL_VERIFY=false"

	combinedVars := httpAddrVar

	if c.ACL {
		combinedVars = strings.Join([]string{httpAddrVar, tokenVar}, "\n")
	}

	if c.Insecure {
		combinedVars = strings.Join([]string{combinedVars, insecureTLSVar}, "\n")
	}

	return combinedVars
}

func (c ClientConfigOpts) getNomadVariables() string {

	if c.Unset {
		return "unset NOMAD_ADDR\nunset NOMAD_TOKEN\nunset NOMAD_SKIP_VERIFY"
	}

	addr := getIPAddress(c.getRandomServer())
	scheme := "http://"
	port := 4646
	bootstrapToken := "00000000-0000-0000-0000-000000000000" // consul bootstrap token

	if c.TLS {
		scheme = "https://"
	}

	nomadHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export NOMAD_ADDR=%s", nomadHTTPAddr)
	tokenVar := fmt.Sprintf("export NOMAD_TOKEN=%s", bootstrapToken)
	insecureTLSVar := "export NOMAD_SKIP_VERIFY=true"

	combinedVars := httpAddrVar

	if c.ACL {
		combinedVars = strings.Join([]string{httpAddrVar, tokenVar}, "\n")
	}

	if c.Insecure {
		combinedVars = strings.Join([]string{combinedVars, insecureTLSVar}, "\n")
	}

	return combinedVars
}

func (c ClientConfigOpts) getVaultVariables() string {

	if c.Unset {
		return "unset VAULT_ADDR\nunset VAULT_SKIP_VERIFY"
	}

	addr := getIPAddress(c.getRandomServer())
	scheme := "http://"
	port := 8200

	if c.TLS {
		scheme = "https://"
	}

	vaultHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export VAULT_ADDR=%s", vaultHTTPAddr)
	insecureTLSVar := "export VAULT_SKIP_VERIFY=true"

	combinedVars := httpAddrVar

	if c.Insecure {
		combinedVars = strings.Join([]string{combinedVars, insecureTLSVar}, "\n")
	}

	return combinedVars
}
