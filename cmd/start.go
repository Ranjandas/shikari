/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts VMs that belong to the named cluster",
	Long: `Starts VMs that belong to the named cluster

Example:

$ shikari start -n murphy`,
	RunE: func(cmd *cobra.Command, args []string) error {
		instances, err := lima.GetClusterInstances(cluster.Name)
		if err != nil {
			return err
		}

		if len(instances) == 0 {
			return fmt.Errorf("cluster %s not found", cluster.Name)
		}

		stoppedInstances := lima.GetInstancesByStatus(instances, "stopped")

		if len(stoppedInstances) == 0 {
			fmt.Printf("No stopped instances in the %s cluster to start.\n", cluster.Name)
			return nil
		}

		// A start that fails is retried once, as it is safe to repeat
		return lima.RunAll(lima.Names(stoppedInstances), lima.Parallelism(), 1, lima.StartLimaVM)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	addNameFlag(startCmd, true)

}
