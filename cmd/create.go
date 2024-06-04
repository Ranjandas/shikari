/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates multiple VMs to fom a cluster.",
	Long: `Creates multiple VMs to for a cluster

For example:

$ shikari create --name murphy --servers 3  --clients 3 --template hashibox --env CONSUL_LICENSE=$(cat consul.hclic)

The above command will create a 3 server and 3 client cluster, each vm
carrying the name as a prefix to easily identify.
`,
	Run: func(cmd *cobra.Command, args []string) {

		serverVMs := generateServerInstanceNames(name, servers)

		userDefinedEnvs := generateEnvArgs(cmd)

		var wg sync.WaitGroup
		errCh := make(chan error, len(serverVMs))

		// Spawn Lima VMs concurrently
		for _, vmName := range serverVMs {
			wg.Add(1)
			go spawnLimaVM(vmName, "server", userDefinedEnvs, &wg, errCh)
			// @TODO - Serialize properly
			time.Sleep(10 * time.Second)
		}

		clientVMs := generateClientInstanceNames(name, clients)
		for _, vmName := range clientVMs {
			wg.Add(1)
			go spawnLimaVM(vmName, "client", userDefinedEnvs, &wg, errCh)
			// @TODO - Serialize properly
			time.Sleep(10 * time.Second)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Close error channel after all goroutines are done
		close(errCh)

		// Check for any errors reported by the goroutines
		for err := range errCh {
			fmt.Println(err)
		}
	},
}

var servers, clients int
var name, template string

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	createCmd.Flags().IntVarP(&servers, "servers", "s", 1, "number of servers")
	createCmd.Flags().IntVarP(&clients, "clients", "c", 0, "number of clients")
	createCmd.Flags().StringVarP(&name, "name", "n", "shikari", "name of the cluster")
	createCmd.Flags().StringVarP(&template, "template", "t", "alpine", "name of lima template for the VMs")
	createCmd.Flags().StringSliceP("env", "e", []string{}, "provide environment vars in the for key=value (can be used multiple times)")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("servers")

}

func spawnLimaVM(vmName string, modeEnv string, userEnv string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	//tmpl := fmt.Sprintf("template://%s", template)

	//--set '. |= .env.mode="server", .env.cluster="murphy"'
	yqExpression := fmt.Sprintf(`.env.CLUSTER="%s" | .env.MODE="%s"`, name, modeEnv)

	// append user defined environment variable
	if userEnv != "" {
		yqExpression = fmt.Sprintf("%s | %s", yqExpression, userEnv)
	}

	// Define the command to spawn a Lima VM
	limaCmd := fmt.Sprintf("limactl start --name %s %s --tty=false --set '%s'", vmName, template, yqExpression)
	cmd := exec.Command("/bin/sh", "-c", limaCmd)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		errCh <- fmt.Errorf("error spawning Lima VM %s: %w", vmName, err)
		return
	}

	fmt.Printf("Lima VM %s spawned successfully.\n", vmName)
}

func generateServerInstanceNames(name string, numServers int) []string {

	s := make([]string, 0)

	for n := 1; n <= numServers; n++ {
		name := fmt.Sprintf("%s-srv-%02d", name, n)

		s = append(s, name)
	}
	return s
}

func generateClientInstanceNames(name string, numClients int) []string {

	s := make([]string, 0)

	for n := 1; n <= numClients; n++ {
		name := fmt.Sprintf("%s-cli-%02d", name, n)

		s = append(s, name)
	}
	return s
}

func generateEnvArgs(cmd *cobra.Command) string {
	envs, _ := cmd.Flags().GetStringSlice("env")

	var envCSV []string
	for _, e := range envs {
		if !strings.Contains(e, "=") {
			fmt.Println("Invalid env format")
			os.Exit(1)
		}

		kv := strings.SplitN(e, "=", 2)

		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			fmt.Println("Invalid env format")
			os.Exit(1)
		}

		envCSV = append(envCSV, fmt.Sprintf(".env.%s=\"%s\"", kv[0], kv[1]))
	}

	return strings.Join(envCSV, "| ")
}
