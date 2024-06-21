/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"
	"os"
	"regexp"
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
	listCmd.Flags().BoolVarP(&header, "no-header", "", false, "skip the header from list output")
}

var header bool

func listInstances(clusterName string) {
	vms := lima.ListInstances()

	w := tabwriter.NewWriter(os.Stdout, 5, 3, 7, byte(' '), 0)

	if !header {
		fmt.Fprintln(w, "CLUSTER\tVM NAME\tSATUS\tDISK(GB)\tMEMORY(GB)\tCPUS")
	}

	for _, vm := range vms {
		if isShikariVM(vm.Name) {

			if len(vm.Name) > 0 {
				if !strings.HasPrefix(vm.Name, clusterName) {
					continue //skip printing the
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\n", getClusterNameFromInstanceName(vm.Name), vm.Name, vm.Status, bytesToGiB(vm.Disk), bytesToGiB(vm.Memory), vm.Cpus)
		}
	}
	w.Flush()
}

func getClusterNameFromInstanceName(name string) string {
	clusterName := strings.Split(name, "-")

	return clusterName[0]
}

func isShikariVM(name string) bool {
	pattern := `^([a-zA-Z]+)-(srv|cli)-(\d+)$`

	regex, err := regexp.Compile(pattern)

	if err != nil {
		fmt.Println("Error compiling regex:", err)
	}

	if match := regex.MatchString(name); match {
		return true
	}

	return false
}

func bytesToGiB(bytes uint64) uint64 {
	const GiB = 1 << (10 * 3)
	return bytes / GiB
}
