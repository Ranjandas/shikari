/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops VM's that belong to the named cluster",
	Long: `Stops VM's that belong to the named cluster

	Exmple:
	
	$ shikari stop -n murphy`,
	RunE: func(cmd *cobra.Command, args []string) error {

		instances, err := lima.GetClusterInstances(cluster.Name)
		if err != nil {
			return err
		}

		if len(instances) == 0 {
			return fmt.Errorf("cluster %s not found", cluster.Name)
		}

		runningInstances := lima.GetInstancesByStatus(instances, "running")

		if len(runningInstances) == 0 {
			fmt.Printf("No running instances in the %s cluster to stop.\n", cluster.Name)
			return nil
		}

		return lima.RunAll(lima.Names(runningInstances), lima.Parallelism(), 0, lima.StopLimaVM)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	addNameFlag(stopCmd, true)

}
