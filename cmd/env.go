/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
shikari env -n murphy --tls consul,nomad
export CONSUL_HTTP_ADDR=https://xxx.xxx
export CONSUL_CACERT=xxx/consul-agent-ca.pem
export NOMAD_ADDR=https://xxx.xxx
export NOMAD_CACERT=xxx/nomad-agent-ca.pem

The first running server is used, unless one is picked with --server:
shikari env -n murphy -s srv-02 nomad`,
	RunE: func(cmd *cobra.Command, args []string) error {

		instances, err := lima.GetClusterInstances(cluster.Name)
		if err != nil {
			return err
		}

		if len(instances) == 0 {
			return fmt.Errorf("cluster \"%s\" not found", cluster.Name)
		}

		if !(len(args) > 0) {
			return fmt.Errorf("no product name passed as argument. Supported product names %v", validProducts)
		}
		// Set ClientConfig Name same as Cluster Name
		clientConfigOpts.Name = cluster.Name
		// List the VMs once, for all the products
		clientConfigOpts.instances = instances
		products := args[0]

		var errs []error
		for _, product := range strings.Split(products, ",") {
			if isValidProduct(product) {
				if err := clientConfigOpts.printClientConfigEnvs(product); err != nil {
					errs = append(errs, err)
				}
			} else {
				errs = append(errs, fmt.Errorf("invalid product name %s. Supported product names %v", product, validProducts))
			}
		}

		return errors.Join(errs...)
	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	addNameFlag(envCmd, true)
	envCmd.Flags().BoolVarP(&clientConfigOpts.ACL, "acl", "a", false, "prints the ACL token variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.TLS, "tls", "t", false, "prints the TLS variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.Insecure, "insecure", "i", false, "prints the skip TLS Verify variables")
	envCmd.Flags().BoolVarP(&clientConfigOpts.Unset, "unset", "u", false, "unset the variables instead of export")
	envCmd.Flags().StringVarP(&clientConfigOpts.Server, "server", "s", "", "server to point at, eg: srv-02 (default: the first running server)")

}

type ClientConfigOpts struct {
	Name     string
	TLS      bool
	ACL      bool
	Insecure bool
	Unset    bool
	Server   string

	instances  []lima.LimaVM
	serverAddr string
	serverEnv  map[string]string
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

func (c *ClientConfigOpts) printClientConfigEnvs(product string) error {
	var vars string
	var err error

	switch product {
	case "consul":
		vars, err = c.getConsulVariables()
	case "nomad":
		vars, err = c.getNomadVariables()
	case "vault":
		vars, err = c.getVaultVariables()
	case "boundary":
		vars, err = c.getBoundaryVariables()
	case "k3s":
		vars, err = c.getK3SVariables()
	}

	if err != nil {
		return fmt.Errorf("%s: %w", product, err)
	}

	fmt.Println(vars)
	return nil
}

// getServerAddress returns the address of the server picked with --server, or
// else of the first running server. The server is looked up once, so all the
// products point at the same one.
func (c *ClientConfigOpts) getServerAddress() (string, error) {
	if c.serverAddr != "" {
		return c.serverAddr, nil
	}

	server, err := pickServer(c.instances, c.Name, c.Server)
	if err != nil {
		return "", err
	}

	addr := server.GetIPAddress()
	if addr == "" {
		return "", fmt.Errorf("could not get the IP address of %s", server.Name)
	}

	c.serverAddr = addr
	c.serverEnv = server.Config.Env
	return addr, nil
}

// setting returns a value the template set in the server's env, eg:
// SHIKARI_CONSUL_TOKEN, or def if it did not. getServerAddress must be called
// first.
func (c *ClientConfigOpts) setting(key string, def string) string {
	if value, ok := c.serverEnv[key]; ok && value != "" {
		return value
	}

	return def
}

// port returns a port the template set in the server's env, or def.
func (c *ClientConfigOpts) port(key string, def int) (int, error) {
	value := c.setting(key, "")
	if value == "" {
		return def, nil
	}

	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s=%q in the template is not a port number", key, value)
	}

	return port, nil
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9._:/@%+=,-]+$`)

// shellQuote quotes value for `eval $(shikari env ...)`, so a token from a
// template can't run commands on the host.
func shellQuote(value string) string {
	if shellSafe.MatchString(value) {
		return value
	}

	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// pickServer returns the named server, eg: srv-02 or murphy-srv-02, or else
// the running server with the lowest number.
func pickServer(instances []lima.LimaVM, clusterName string, server string) (lima.LimaVM, error) {
	servers := lima.GetServerInstances(instances)
	sort.Slice(servers, func(i, j int) bool { return servers[i].Index() < servers[j].Index() })

	if server != "" {
		name := server
		if !strings.HasPrefix(name, clusterName+"-") {
			name = clusterName + "-" + server
		}

		for _, vm := range servers {
			if vm.Name != name {
				continue
			}
			if !strings.EqualFold(vm.Status, "running") {
				return lima.LimaVM{}, fmt.Errorf("server %s is %s, start it with: shikari start -n %s", name, strings.ToLower(vm.Status), clusterName)
			}
			return vm, nil
		}

		return lima.LimaVM{}, fmt.Errorf("server %s not found in the cluster %s, the servers are: %s", name, clusterName, strings.Join(lima.Names(servers), ", "))
	}

	for _, vm := range servers {
		if strings.EqualFold(vm.Status, "running") {
			return vm, nil
		}
	}

	if len(servers) == 0 {
		return lima.LimaVM{}, fmt.Errorf("the cluster %s has no servers", clusterName)
	}

	return lima.LimaVM{}, fmt.Errorf("there are no running servers in the cluster %s (%s), start them with: shikari start -n %s", clusterName, strings.Join(notRunning(servers), ", "), clusterName)
}

// notRunning returns "name (status)" for every instance, eg: murphy-srv-01 (stopped).
func notRunning(instances []lima.LimaVM) []string {
	var states []string
	for _, vm := range instances {
		states = append(states, fmt.Sprintf("%s (%s)", vm.Name, strings.ToLower(vm.Status)))
	}

	return states
}

// firstServer returns the cluster's <name>-srv-01 VM.
func (c *ClientConfigOpts) firstServer() (lima.LimaVM, bool) {
	name := fmt.Sprintf("%s-srv-01", c.Name)

	for _, vm := range c.instances {
		if vm.Name == name {
			return vm, true
		}
	}

	return lima.LimaVM{}, false
}

func (c *ClientConfigOpts) getConsulVariables() (string, error) {

	if c.Unset {
		return "unset CONSUL_HTTP_ADDR\nunset CONSUL_HTTP_TOKEN\nunset CONSUL_HTTP_SSL_VERIFY\nunset CONSUL_CACERT", nil
	}

	addr, err := c.getServerAddress()
	if err != nil {
		return "", err
	}
	scheme := "http://"
	port, err := c.port("SHIKARI_CONSUL_HTTP_PORT", 8500)
	if err != nil {
		return "", err
	}
	bootstrapToken := c.setting("SHIKARI_CONSUL_TOKEN", "root") // consul bootstrap token
	caCertVar := ""

	if c.TLS {
		scheme = "https://"
		port, err = c.port("SHIKARI_CONSUL_HTTPS_PORT", 8501)
		if err != nil {
			return "", err
		}
		caCertVar = fmt.Sprintf("export CONSUL_CACERT=%s", c.getTLSCaCertPath("consul"))
	}

	consulHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export CONSUL_HTTP_ADDR=%s", consulHTTPAddr)
	tokenVar := fmt.Sprintf("export CONSUL_HTTP_TOKEN=%s", shellQuote(bootstrapToken))
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

	return combinedVars, nil
}

func (c *ClientConfigOpts) getNomadVariables() (string, error) {

	if c.Unset {
		return "unset NOMAD_ADDR\nunset NOMAD_TOKEN\nunset NOMAD_SKIP_VERIFY\nunset NOMAD_CACERT", nil
	}

	addr, err := c.getServerAddress()
	if err != nil {
		return "", err
	}
	scheme := "http://"
	port, err := c.port("SHIKARI_NOMAD_PORT", 4646)
	if err != nil {
		return "", err
	}
	bootstrapToken := c.setting("SHIKARI_NOMAD_TOKEN", "00000000-0000-0000-0000-000000000000") // nomad bootstrap token
	caCertVar := ""

	if c.TLS {
		scheme = "https://"
		caCertVar = fmt.Sprintf("export NOMAD_CACERT=%s", c.getTLSCaCertPath("nomad"))
	}

	nomadHTTPAddr := fmt.Sprintf("%s%s:%d", scheme, addr, port)

	// env-variable=value
	httpAddrVar := fmt.Sprintf("export NOMAD_ADDR=%s", nomadHTTPAddr)
	tokenVar := fmt.Sprintf("export NOMAD_TOKEN=%s", shellQuote(bootstrapToken))
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

	return combinedVars, nil
}

func (c *ClientConfigOpts) getVaultVariables() (string, error) {

	if c.Unset {
		return "unset VAULT_ADDR\nunset VAULT_SKIP_VERIFY\nunset VAULT_CACERT", nil
	}

	addr, err := c.getServerAddress()
	if err != nil {
		return "", err
	}
	scheme := "http://"
	port, err := c.port("SHIKARI_VAULT_PORT", 8200)
	if err != nil {
		return "", err
	}
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

	return combinedVars, nil
}

func (c *ClientConfigOpts) getBoundaryVariables() (string, error) {
	// ref: https://developer.hashicorp.com/boundary/docs/commands#environment-variables

	if c.Unset {
		return "unset BOUNDARY_ADDR\nunset BOUNDARY_TLS_INSECURE\nunset BOUNDARY_CACERT", nil
	}

	addr, err := c.getServerAddress()
	if err != nil {
		return "", err
	}
	scheme := "http://"
	port, err := c.port("SHIKARI_BOUNDARY_PORT", 9200)
	if err != nil {
		return "", err
	}
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

	return combinedVars, nil
}

func (c *ClientConfigOpts) getK3SVariables() (string, error) {
	var k3sKubeConfig string

	vm, ok := c.firstServer()
	if !ok {
		return "", fmt.Errorf("instance %s-srv-01 not found", c.Name)
	}

	err := c.copyK3SKubeConfig(vm)
	if err != nil {
		return "", fmt.Errorf("error copying KUBECONFIG for cluster %s: %w", c.Name, err)
	}

	k3sKubeConfig = fmt.Sprintf("export KUBECONFIG=%s/k3s.yaml", vm.Dir)

	return k3sKubeConfig, nil
}

func (c *ClientConfigOpts) copyK3SKubeConfig(vm lima.LimaVM) error {

	cmd := exec.Command("limactl", "copy", vm.Name+":/etc/rancher/k3s/k3s.yaml", vm.Dir)

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

func (c *ClientConfigOpts) getTLSCaCertPath(product string) string {
	vm, ok := c.firstServer()
	if !ok {
		return "" //There are no VMs in the cluster
	}

	return fmt.Sprintf("%s/copied-from-guest/%s-agent-ca.pem", vm.GetVMDir(), product)
}
