/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List VMs belonging to clusters",
	Long:  `List VMs belonging to clusters`,
	Run: func(cmd *cobra.Command, args []string) {
		listInstances(cluster.Name)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the  cluster")
	listCmd.Flags().BoolVarP(&noheader, "no-header", "", false, "skip the header from list output")
}

var noheader bool

func listInstances(clusterName string) {
	vms := lima.ListInstances()

	w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, byte(' '), 0)

	if !noheader {
		fmt.Fprintln(w, "CLUSTER\tVM NAME\tARCH\tIP(lima0)\tSTATUS\tSCENARIO\tDISK(GB)\tMEMORY(GB)\tCPUS\tIMAGE")
	}

	if clusterName != "" {
		vms = lima.FilterClusterInstances(vms, clusterName)
	} else {
		vms = lima.FilterShikariInstances(vms)
	}

	for _, vm := range vms {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%s\n", vm.ClusterName(),
			vm.Name, vm.Arch, vm.GetIPAddress(),
			vm.Status, vm.GetScenarioNameFromEnv(),
			bytesToGiB(vm.Disk), bytesToGiB(vm.Memory),
			vm.Cpus,
			getImageLocation(vm),
		)
	}
	w.Flush()
}

func getImageLocation(vm lima.LimaVM) string {

	var location string

	for _, image := range vm.Config.Images {

		if image.Arch == vm.Arch {
			location = image.Location
		}
	}

	// If location is empty, return it as-is
	if location == "" {
		return location
	}

	// Expand tilde to home directory if present
	expandedPath := location
	if strings.HasPrefix(location, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			expandedPath = filepath.Join(homeDir, location[2:])
		}
	}

	// Check if the expanded location is a symlink and resolve to absolute path
	fileInfo, err := os.Lstat(expandedPath)
	if err != nil {
		// If we can't stat the file, return the original location
		return location
	}

	// If it's a symlink, resolve it to the absolute path
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		resolvedPath, err := filepath.EvalSymlinks(expandedPath)
		if err != nil {
			// If we can't resolve the symlink, return the original location
			return location
		}
		return resolvedPath
	}

	return location
}

func bytesToGiB(bytes uint64) uint64 {
	const GiB = 1 << (10 * 3)
	return bytes / GiB
}
