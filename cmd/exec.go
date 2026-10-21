/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"errors"
	"fmt"
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
	RunE: func(cmd *cobra.Command, args []string) error {

		quiet, _ := cmd.Flags().GetBool("quiet")
		execAll, _ := cmd.Flags().GetBool("all")
		execServers, _ := cmd.Flags().GetBool("servers")
		execClients, _ := cmd.Flags().GetBool("clients")
		execInstance, _ := cmd.Flags().GetString("instance")

		clusterName, _ := cmd.Flags().GetString("name")

		if len(args) == 0 {
			return fmt.Errorf("no commands provided as args to execute")
		}

		clusterInstances, err := lima.GetClusterInstances(clusterName)
		if err != nil {
			return err
		}

		instances := lima.GetInstancesByStatus(clusterInstances, "running")

		if len(instances) == 0 {
			return fmt.Errorf("there are no running instances in the cluster %s", clusterName)
		}

		// keep running against the remaining VMs when one fails, and fail at the end
		var errs []error
		execVM := func(vmName string) {
			if err := lima.ExecLimaVM(vmName, strings.Join(args, " "), quiet); err != nil {
				errs = append(errs, err)
			}
		}

		if execAll {
			for _, vmName := range instances {
				execVM(vmName.Name)
			}
		}

		if execServers {
			for _, vmName := range instances {
				if strings.HasPrefix(vmName.Name, fmt.Sprintf("%s-srv", clusterName)) {
					execVM(vmName.Name)
				}

			}
		}

		if execClients {
			for _, vmName := range instances {
				if strings.HasPrefix(vmName.Name, fmt.Sprintf("%s-cli", clusterName)) {
					execVM(vmName.Name)
				}

			}
		}

		var instanceExists bool
		if execInstance != "" {
			for _, vmName := range instances {
				if strings.HasSuffix(vmName.Name, execInstance) {
					instanceExists = true
					execVM(vmName.Name)
				}
			}
			if !instanceExists {
				errs = append(errs, fmt.Errorf("no instance matching the name *%s exists in cluster %s", execInstance, clusterName))
			}
		}

		return errors.Join(errs...)
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
