/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sync"
	"time"

	lima "github.com/ranjandas/shikari/app"
	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroys VM's that belong to the named cluster",
	Long: `Destroys VM's that belong to the named cluster

	Exmple:
	
	$ shikari destroy -n murphy`,
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Println("destroy called")

		instances := lima.GetInstancesByPrefix(name)
		runningInstances := lima.GetInstancesByStatus(instances, "running")

		if len(runningInstances) != 0 {
			fmt.Println("There are running instances in the cluster, cannot destroy!")
			return
		}

		stoppedInstances := lima.GetInstancesByStatus(instances, "stopped")

		var wg sync.WaitGroup
		errCh := make(chan error, len(stoppedInstances))

		// Stop Lima VMs concurrently
		for _, vmName := range stoppedInstances {
			wg.Add(1)
			go lima.DeleteLimaVM(vmName.Name, force, &wg, errCh)
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

var force bool // whether to force delete the vm

func init() {
	rootCmd.AddCommand(destroyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// destroyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// destroyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	destroyCmd.Flags().StringVarP(&name, "name", "n", "shikari", "name of the cluster")
	destroyCmd.Flags().BoolVarP(&force, "force", "f", false, "force destruction of the cluster even when VMs are running")
}
