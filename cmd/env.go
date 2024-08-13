/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

var validProducts = []string{"consul", "nomad", "vault", "boundary", "k3s"}

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Prints client config environment variables",
	Long: `Prints client config environment variables.
Multiple product names can be passed separated by comma.

For example:
shikari env -n murphy -t consul,nomad
export CONSUL_HTTP_ADDR=https://xxx.xxx
export CONSUL_CACERT=xxx/consul-agent-ca.pem
export NOMAD_ADDR=https://xxx.xxx
export NOMAD_CACERT=xxx/nomad-agent-ca.pem`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(lima.GetInstancesByPrefix(cluster.Name)) == 0 {
			fmt.Printf("Cluster \"%s\" not found.\n", cluster.Name)
			return
		}

		if !(len(args) > 0) {
			fmt.Println("No product name passed as argument. Supported product names", validProducts)
			return
		}
		// Set ClientConfig Name same as Cluster Name
		// TODO: Refcator the flags by unifying common flags
		clientConfigOpts.Name = cluster.Name
		products := args[0]

		for _, product := range strings.Split(products, ",") {
			if isValidProduct(product) {
				clientConfigOpts.printClientConfigEnvs(product)
			} else {
				fmt.Println("\nInvalid product name", product)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	envCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	envCmd.Flags().BoolVarP(&clientConfigOpts.ACL, "acl", "a", false, "prints the ACL token variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.TLS, "tls", "t", false, "prints the TLS variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.Insecure, "insecure", "i", false, "prints the skip TLS Verify variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.Unset, "unset", "u", false, "unset the variables insetad of export")

	envCmd.MarkFlagRequired("name")
}

type ClientConfigOpts struct {
	Name     string
	TLS      bool
	ACL      bool
	Insecure bool
	Unset    bool
}

var clientConfigOpts ClientConfigOpts

func isValidProduct(product string) bool {
	for _, p := range validProducts {
		if product == p {
			return true
		}
	}
	return false
}

func (c ClientConfigOpts) printClientConfigEnvs(product string) {
	switch product {
	case "consul":
		fmt.Println(c.getConsulVariables())
	case "nomad":
		fmt.Println(c.getNomadVariables())
	case "vault":
		fmt.Println(c.getVaultVariables())
	case "boundary":
		fmt.Println(c.getBoundaryVariables())
	case "k3s":
		fmt.Println(c.getK3SVariables())
	}
}

func (c ClientConfigOpts) getRandomServer() lima.LimaVM {
	// always get the instances of type server "-srv"
	instances := lima.GetInstancesByPrefix(fmt.Sprintf("%s-srv", c.Name))
	runningInstances := lima.GetInstancesByStatus(instances, "running")

	if !(len(runningInstances) > 0) {
		os.Exit(1)
	}

	randomIndex := rand.Intn(len(runningInstances))
	return runningInstances[randomIndex]
}

func (c ClientConfigOpts) getConsulVariables() string {

	if c.Unset {
		return "unset CONSUL_HTTP_ADDR\nunset CONSUL_HTTP_TOKEN\nunset CONSUL_HTTP_SSL_VERIFY\nunset CONSUL_CACERT"
	}

	addr := c.getRandomServer().GetIPAddress()
	scheme := "http://"
	port := 8500
	bootstrapToken := "root" // consul bootstrap token
	caCertVar := ""

	if c.TLS {
		scheme = "https://"
		port = 8501
		caCertVar = fmt.Sprintf("export CONSUL_CACERT=%s", c.getTLSCaCertPath("consul"))
	}

	consulHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export CONSUL_HTTP_ADDR=%s", consulHTTPAddr)
	tokenVar := fmt.Sprintf("export CONSUL_HTTP_TOKEN=%s", bootstrapToken)
	insecureTLSVar := "export CONSUL_HTTP_SSL_VERIFY=false"

	combinedVars := httpAddrVar

	if c.TLS {
		combinedVars = strings.Join([]string{combinedVars, caCertVar}, "\n")
	}

	if c.ACL {
		combinedVars = strings.Join([]string{combinedVars, tokenVar}, "\n")
	}

	if c.Insecure {
		combinedVars = strings.Join([]string{combinedVars, insecureTLSVar}, "\n")
	}

	return combinedVars
}

func (c ClientConfigOpts) getNomadVariables() string {

	if c.Unset {
		return "unset NOMAD_ADDR\nunset NOMAD_TOKEN\nunset NOMAD_SKIP_VERIFY\nunset NOMAD_CACERT"
	}

	addr := c.getRandomServer().GetIPAddress()
	scheme := "http://"
	port := 4646
	bootstrapToken := "00000000-0000-0000-0000-000000000000" // consul bootstrap token
	caCertVar := ""

	if c.TLS {
		scheme = "https://"
		caCertVar = fmt.Sprintf("export NOMAD_CACERT=%s", c.getTLSCaCertPath("nomad"))
	}

	nomadHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export NOMAD_ADDR=%s", nomadHTTPAddr)
	tokenVar := fmt.Sprintf("export NOMAD_TOKEN=%s", bootstrapToken)
	insecureTLSVar := "export NOMAD_SKIP_VERIFY=true"

	combinedVars := httpAddrVar

	if c.TLS {
		combinedVars = strings.Join([]string{combinedVars, caCertVar}, "\n")
	}

	if c.ACL {
		combinedVars = strings.Join([]string{combinedVars, tokenVar}, "\n")
	}

	if c.Insecure {
		combinedVars = strings.Join([]string{combinedVars, insecureTLSVar}, "\n")
	}

	return combinedVars
}

func (c ClientConfigOpts) getVaultVariables() string {

	if c.Unset {
		return "unset VAULT_ADDR\nunset VAULT_SKIP_VERIFY\nunset VAULT_CACERT"
	}

	addr := c.getRandomServer().GetIPAddress()
	scheme := "http://"
	port := 8200
	caCertVar := ""

	if c.TLS {
		scheme = "https://"
		caCertVar = fmt.Sprintf("export VAULT_CACERT=%s", c.getTLSCaCertPath("vault"))
	}

	vaultHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export VAULT_ADDR=%s", vaultHTTPAddr)
	insecureTLSVar := "export VAULT_SKIP_VERIFY=true"

	combinedVars := httpAddrVar

	if c.TLS {
		combinedVars = strings.Join([]string{combinedVars, caCertVar}, "\n")
	}

	if c.Insecure {
		combinedVars = strings.Join([]string{combinedVars, insecureTLSVar}, "\n")
	}

	return combinedVars
}

func (c ClientConfigOpts) getBoundaryVariables() string {
	// ref: https://developer.hashicorp.com/boundary/docs/commands#environment-variables

	if c.Unset {
		return "unset BOUNDARY_ADDR\nunset BOUNDARY_TLS_INSECURE\nunset BOUNDARY_CACERT"
	}

	addr := c.getRandomServer().GetIPAddress()
	scheme := "http://"
	port := 9200
	caCertVar := ""

	if c.TLS {
		scheme = "https://"
		caCertVar = fmt.Sprintf("export BOUNDARY_CACERT=%s", c.getTLSCaCertPath("boundary"))
	}

	boundaryHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export BOUNDARY_ADDR=%s", boundaryHTTPAddr)
	insecureTLSVar := "export BOUNDARY_TLS_INSECURE=true"

	combinedVars := httpAddrVar

	if c.TLS {
		combinedVars = strings.Join([]string{combinedVars, caCertVar}, "\n")
	}

	if c.Insecure {
		combinedVars = strings.Join([]string{combinedVars, insecureTLSVar}, "\n")
	}

	return combinedVars
}

func (c ClientConfigOpts) getK3SVariables() string {
	var k3sKubeConfig string

	vm := lima.GetInstance(fmt.Sprintf("%s-srv-01", c.Name))

	if vm.Name == "" {
		return "" //There are no VMs in the cluster
	}

	err := c.copyK3SKubeConfig()
	if err != nil {
		fmt.Printf("error copying KUBECONFIG for Cluster: %s %v\n", c.Name, err)
		os.Exit(1)
	}

	k3sKubeConfig = fmt.Sprintf("export KUBECONFIG=%s/k3s.yaml", vm.Dir)

	return k3sKubeConfig
}

func (c ClientConfigOpts) copyK3SKubeConfig() error {

	vm := lima.GetInstance(fmt.Sprintf("%s-srv-01", c.Name))

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("limactl copy %s:/etc/rancher/k3s/k3s.yaml %s", vm.Name, vm.Dir))

	// Run the command
	err := cmd.Run()
	if err != nil {
		return err
	}

	// Set 0600 permission as a best practice
	filepath := filepath.Join(vm.Dir, "k3s.yaml")

	err = os.Chmod(filepath, 0600)

	return err
}

func (c ClientConfigOpts) getTLSCaCertPath(product string) string {
	firstInstance := fmt.Sprintf("%s-srv-01", c.Name)

	vm := lima.GetInstance(firstInstance)

	if vm.Name == "" {
		return "" //There are no VMs in the cluster
	}

	return fmt.Sprintf("%s/copied-from-guest/%s-agent-ca.pem", vm.GetVMDir(), product)
}
