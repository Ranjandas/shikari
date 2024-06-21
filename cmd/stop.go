/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sync"

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
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Println("stop called")

		instances := lima.GetInstancesByPrefix(cluster.Name)
		runningInstances := lima.GetInstancesByStatus(instances, "running")

		if len(runningInstances) == 0 {
			fmt.Printf("No stopped instances in the %s cluster to stop.\n", cluster.Name)
			return
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(runningInstances))

		// Stop Lima VMs concurrently
		for _, vmName := range runningInstances {
			wg.Add(1)
			go lima.StopLimaVM(vmName.Name, &wg, errCh)
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
	rootCmd.AddCommand(stopCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stopCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stopCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	stopCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	stopCmd.MarkFlagRequired("name")

}
