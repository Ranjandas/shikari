/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroys VM's that belong to the named cluster",
	Long: `Destroys VM's that belong to the named cluster

	Exmple:

	$ shikari destroy -n murphy`,
	RunE: func(cmd *cobra.Command, args []string) error {
		allInstances, err := lima.GetClusterInstances(cluster.Name)
		if err != nil {
			return err
		}

		if len(allInstances) == 0 {
			return fmt.Errorf("no instances in the cluster %s", cluster.Name)
		}

		if cluster.Force {
			return destroyVM(allInstances, true)
		}

		if blocking := notStopped(allInstances); len(blocking) > 0 {
			return fmt.Errorf("cannot destroy cluster %s, these instances are not stopped: %s. Stop them or rerun the command with -f", cluster.Name, strings.Join(blocking, ", "))
		}

		return destroyVM(allInstances, false)
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	addNameFlag(destroyCmd, true)
	destroyCmd.Flags().BoolVarP(&cluster.Force, "force", "f", false, "force destruction of the cluster even when VMs are running")
}

// notStopped returns "name (status)" for every instance that is not stopped,
// eg: running or broken VMs, which block a destroy without -f.
func notStopped(instances []lima.LimaVM) []string {
	var blocking []string

	for _, instance := range instances {
		if strings.ToLower(instance.Status) != "stopped" {
			blocking = append(blocking, fmt.Sprintf("%s (%s)", instance.Name, instance.Status))
		}
	}

	return blocking
}

func destroyVM(instances []lima.LimaVM, force bool) error {
	// A delete that fails is retried once, as it is safe to repeat
	return lima.RunAll(lima.Names(instances), lima.Parallelism(), 1, func(vmName string) error {
		return lima.DeleteLimaVM(vmName, force)
	})
}
