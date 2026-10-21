package cmd

import (
	"github.com/spf13/cobra"
)

// addNameFlag adds --name/-n, the cluster the command works on, to cmd. It is
// defined here once, rather than as a persistent flag, as shell, license and
// version don't take a cluster.
func addNameFlag(cmd *cobra.Command, required bool) {
	usage := "name of the cluster"
	if !required {
		usage = "name of the cluster (default: all clusters)"
	}

	cmd.Flags().StringVarP(&cluster.Name, "name", "n", "", usage)

	if required {
		cmd.MarkFlagRequired("name")
	}
}
