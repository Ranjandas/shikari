/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sync"
	"time"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts VM's that belong to the named cluster",
	Long: `Starts VM's that belong to the named cluster

Exmple:

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

		var wg sync.WaitGroup
		errCh := make(chan error, len(stoppedInstances))

		// Stop Lima VMs concurrently
		for _, vmName := range stoppedInstances {
			wg.Add(1)
			go lima.StartLimaVM(vmName.Name, &wg, errCh)
			// @TODO - Serialize properly
			time.Sleep(10 * time.Second)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		return collectErrors(errCh)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	startCmd.MarkFlagRequired("name")

}
