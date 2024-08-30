/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute commands inside the VMs",
	Long: `Execute commands inside the VMs. For example:

You can run commands against specific class of servers (clients, servers or all)`,
	Run: func(cmd *cobra.Command, args []string) {

		quiet, _ := cmd.Flags().GetBool("quiet")
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
				lima.ExecLimaVM(vmName.Name, strings.Join(args, " "), quiet)
			}
		}

		if execServers {
			for _, vmName := range instances {
				if strings.HasPrefix(vmName.Name, fmt.Sprintf("%s-srv", clusterName)) {
					lima.ExecLimaVM(vmName.Name, strings.Join(args, " "), quiet)
				}

			}
		}

		if execClients {
			for _, vmName := range instances {
				if strings.HasPrefix(vmName.Name, fmt.Sprintf("%s-cli", clusterName)) {
					lima.ExecLimaVM(vmName.Name, strings.Join(args, " "), quiet)
				}

			}
		}

		var instanceExists bool
		if execInstance != "" {
			for _, vmName := range instances {
				if strings.HasSuffix(vmName.Name, execInstance) {
					instanceExists = true
					lima.ExecLimaVM(vmName.Name, strings.Join(args, " "), quiet)
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

	execCmd.Flags().BoolP("quiet", "q", false, "prints only the output of the commands without headers")
	execCmd.Flags().BoolP("clients", "c", false, "run commands against client instances in the cluster")
	execCmd.Flags().BoolP("servers", "s", false, "run commands against server instances in the cluster")
	execCmd.Flags().BoolP("all", "a", false, "run commands against all instances in the cluster")
	execCmd.Flags().StringP("instance", "i", "", "name of the specific instance to run the command against")
	execCmd.Flags().StringP("name", "n", "", "name of the cluster to run the command against")

	execCmd.MarkFlagsMutuallyExclusive("clients", "servers", "all", "instance")

}
