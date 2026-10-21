/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"
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

	destroyCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	destroyCmd.Flags().BoolVarP(&cluster.Force, "force", "f", false, "force destruction of the cluster even when VMs are running")
	destroyCmd.MarkFlagRequired("name")
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

	return collectErrors(errCh)
}
