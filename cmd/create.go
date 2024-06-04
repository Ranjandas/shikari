/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

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
		clientVMs := generateClientInstanceNames(name, clients)

		totalVMs := append(serverVMs, clientVMs...)

		userDefinedEnvs := generateEnvArgs(cmd)

		// Validate the Image path and type
		if len(imagePath) > 0 {
			absolutePath, err := filepath.Abs(imagePath)
			if err != nil {
				fmt.Printf("EROR: Cannot find the absolute path of the image: %s", imagePath)
				return
			}

			qcow2, _ := isQCOW2(absolutePath)

			if !qcow2 {
				fmt.Printf("Error: Image %s is not of type qCOW2\n", absolutePath)
				return
			}
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(totalVMs))

		// Spawn Lima VMs concurrently
		for _, vmName := range totalVMs {
			wg.Add(1)
			go spawnLimaVM(vmName, getInstanceMode(vmName), userDefinedEnvs, &wg, errCh)

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
var name, template, imagePath string

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
	createCmd.Flags().IntVarP(&clients, "clients", "c", 1, "number of clients")
	createCmd.Flags().StringVarP(&name, "name", "n", "shikari", "name of the cluster")
	createCmd.Flags().StringVarP(&template, "template", "t", "./hashibox.yaml", "name of lima template for the VMs")
	createCmd.Flags().StringSliceP("env", "e", []string{}, "provide environment vars in the for key=value (can be used multiple times)")
	createCmd.Flags().StringVarP(&imagePath, "image", "i", "", "path to the cqow2 images to be used for the VMs, overriding the one in the template")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("servers")
	createCmd.MarkFlagRequired("clients")

}

func spawnLimaVM(vmName string, modeEnv string, userEnv string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	var tmpl string

	if strings.HasSuffix(strings.ToLower(template), ".yml") || strings.HasSuffix(strings.ToLower(template), ".yaml") {
		tmpl = template
	} else {
		tmpl = fmt.Sprintf("template://%s", template)
	}

	//--set '. |= .env.mode="server", .env.cluster="murphy"'
	yqExpression := fmt.Sprintf(`.env.CLUSTER="%s" | .env.MODE="%s" | .env.BOOTSTRAP_EXPECT="%d"`, name, modeEnv, servers)

	// append user defined environment variable
	if userEnv != "" {
		yqExpression = fmt.Sprintf("%s | %s", yqExpression, userEnv)
	}

	// Override the image from the template
	if len(imagePath) > 0 {
		absolutePath, _ := filepath.Abs(imagePath)
		imageArg := fmt.Sprintf(`.images=[{"location": "%s"}]`, absolutePath)
		yqExpression = fmt.Sprintf("%s | %s", yqExpression, imageArg)
	}

	// Define the command to spawn a Lima VM
	limaCmd := fmt.Sprintf("limactl start --name %s %s --tty=false --set '%s'", vmName, tmpl, yqExpression)
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

func getInstanceMode(instanceName string) string {
	mode := "server"

	if strings.HasPrefix(instanceName, fmt.Sprintf("%s-cli-", name)) {
		mode = "client"
	}

	return mode
}

func isQCOW2(filePath string) (bool, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read the first 4 bytes
	header := make([]byte, 4)
	_, err = file.Read(header)
	if err != nil {
		return false, err
	}

	// Check if the header matches the QCOW2 magic number
	qcow2Magic := []byte{'Q', 'F', 'I', 0xfb}
	if string(header) == string(qcow2Magic) {
		return true, nil
	}

	return false, nil
}
