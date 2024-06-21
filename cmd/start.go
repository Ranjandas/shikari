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
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Println("start called")
		instances := lima.GetInstancesByPrefix(cluster.Name)
		stoppedInstances := lima.GetInstancesByStatus(instances, "stopped")

		if len(stoppedInstances) == 0 {
			fmt.Printf("No stopped instances in the %s cluster to start.\n", cluster.Name)
			return
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

		// Close error channel after all goroutines are done
		close(errCh)

		// Check for any errors reported by the goroutines
		for err := range errCh {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	startCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	startCmd.MarkFlagRequired("name")

}
