/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		return listInstances(os.Stdout, cluster.Name, listOutput)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	addNameFlag(listCmd, false)
	listCmd.Flags().BoolVarP(&noheader, "no-header", "", false, "skip the header from list output")
	listCmd.Flags().StringVarP(&listOutput, "output", "o", "table", "output format: table or json")
}

var noheader bool
var listOutput string

// listedVM is a VM in the json output of list.
type listedVM struct {
	Cluster     string `json:"cluster"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Arch        string `json:"arch"`
	IP          string `json:"ip"`
	Status      string `json:"status"`
	Scenario    string `json:"scenario"`
	DiskBytes   uint64 `json:"disk_bytes"`
	MemoryBytes uint64 `json:"memory_bytes"`
	CPUs        int    `json:"cpus"`
	Image       string `json:"image"`
}

func listInstances(out io.Writer, clusterName string, output string) error {
	if output != "table" && output != "json" {
		return fmt.Errorf("unknown output format %q, use table or json", output)
	}

	vms, err := lima.ListInstances()
	if err != nil {
		return err
	}

	if clusterName != "" {
		vms = lima.FilterClusterInstances(vms, clusterName)
	} else {
		vms = lima.FilterShikariInstances(vms)
	}

	addresses := lima.GetIPAddresses(vms)

	if output == "json" {
		return listJSON(out, vms, addresses)
	}

	w := tabwriter.NewWriter(out, 4, 8, 4, byte(' '), 0)

	if !noheader {
		fmt.Fprintln(w, "CLUSTER\tVM NAME\tARCH\tIP\tSTATUS\tSCENARIO\tDISK(GB)\tMEMORY(GB)\tCPUS\tIMAGE")
	}

	for _, vm := range vms {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%s\n", vm.ClusterName(),
			vm.Name, vm.Arch, addresses[vm.Name],
			vm.Status, vm.GetScenarioNameFromEnv(),
			bytesToGiB(vm.Disk), bytesToGiB(vm.Memory),
			vm.Cpus,
			getImageLocation(vm),
		)
	}
	return w.Flush()
}

func listJSON(out io.Writer, vms []lima.LimaVM, addresses map[string]string) error {
	listed := make([]listedVM, 0, len(vms))

	for _, vm := range vms {
		role := "client"
		if vm.IsServer() {
			role = "server"
		}

		listed = append(listed, listedVM{
			Cluster:     vm.ClusterName(),
			Name:        vm.Name,
			Role:        role,
			Arch:        vm.Arch,
			IP:          addresses[vm.Name],
			Status:      vm.Status,
			Scenario:    vm.GetScenarioNameFromEnv(),
			DiskBytes:   vm.Disk,
			MemoryBytes: vm.Memory,
			CPUs:        vm.Cpus,
			Image:       getImageLocation(vm),
		})
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(listed)
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
