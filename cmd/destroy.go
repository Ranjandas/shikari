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

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroys VM's that belong to the named cluster",
	Long: `Destroys VM's that belong to the named cluster

	Exmple:
	
	$ shikari destroy -n murphy`,
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Println("destroy called")

		allInstances := lima.GetInstancesByPrefix(cluster.Name)

		if len(allInstances) == 0 {
			fmt.Printf("No instances in the cluster %s\n", cluster.Name)
			return
		}

		if cluster.Force {
			destroyVM(allInstances, true)
			return
		}

		runningInstances := lima.GetInstancesByStatus(allInstances, "running")
		if len(runningInstances) > 0 {
			fmt.Println("There are running instances in the cluster, cannot destroy!")
			return
		}

		stoppedInstances := lima.GetInstancesByStatus(allInstances, "stopped")
		if len(allInstances) == len(stoppedInstances) {
			destroyVM(allInstances, false)
		}
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	destroyCmd.Flags().BoolVarP(&cluster.Force, "force", "f", false, "force destruction of the cluster even when VMs are running")
}

func destroyVM(instances []lima.LimaVM, force bool) {
	var wg sync.WaitGroup
	errCh := make(chan error, len(instances))

	// Stop Lima VMs concurrently
	for _, vmName := range instances {
		wg.Add(1)

		go lima.DeleteLimaVM(vmName.Name, force, &wg, errCh)
		time.Sleep(2 * time.Second) //delay the goroutine to avoid errors
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Close error channel after all goroutines are done
	close(errCh)

	// Check for any errors reported by the goroutines
	for err := range errCh {
		fmt.Println(err)
	}
}
