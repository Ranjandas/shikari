/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	lima "github.com/ranjandas/shikari/app"
	"github.com/spf13/cobra"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute commands inside the VMs",
	Long: `Execute commands inside the VMs. For example:

You can run commands against specific class of servers (clients, servers or all)`,
	Run: func(cmd *cobra.Command, args []string) {

		execAll, _ := cmd.Flags().GetBool("all")
		execServers, _ := cmd.Flags().GetBool("servers")
		execClients, _ := cmd.Flags().GetBool("clients")
		execInstance, _ := cmd.Flags().GetString("instance")

		clusterName, _ := cmd.Flags().GetString("name")

		instances := lima.GetInstancesByStatus(lima.GetInstancesByPrefix(clusterName), "running")

		if len(instances) == 0 {
			fmt.Printf("There are no running instances in the cluster %s.\n", clusterName)
		}

		if len(args) == 0 {
			fmt.Println("No commands provided as args to execute. Exiting!")
			os.Exit(0)
		}

		if execAll {
			for _, vmName := range instances {
				lima.ExecLimaVM(vmName.Name, strings.Join(args, " "))
			}
		}

		if execServers {
			for _, vmName := range instances {
				if strings.HasPrefix(vmName.Name, fmt.Sprintf("%s-srv", clusterName)) {
					lima.ExecLimaVM(vmName.Name, strings.Join(args, " "))
				}

			}
		}

		if execClients {
			for _, vmName := range instances {
				if strings.HasPrefix(vmName.Name, fmt.Sprintf("%s-cli", clusterName)) {
					lima.ExecLimaVM(vmName.Name, strings.Join(args, " "))
				}

			}
		}

		var instanceExists bool
		if execInstance != "" {
			for _, vmName := range instances {
				if strings.HasSuffix(vmName.Name, execInstance) {
					instanceExists = true
					lima.ExecLimaVM(vmName.Name, strings.Join(args, " "))
				}
			}
			if !instanceExists {
				fmt.Printf("No instance matching the name *%s exists in cluster %s!\n", execInstance, clusterName)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(execCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// execCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// execCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	execCmd.Flags().BoolP("clients", "c", false, "run commands against client instances in the cluster")
	execCmd.Flags().BoolP("servers", "s", false, "run commands against server instances in the cluster")
	execCmd.Flags().BoolP("all", "a", false, "run commands against all instances in the cluster")
	execCmd.Flags().StringP("instance", "i", "", "name of the specific instance to run the command against")
	execCmd.Flags().StringP("name", "n", "", "name of the cluster to run the command against")

	execCmd.MarkFlagsMutuallyExclusive("clients", "servers", "all", "instance")

}
