/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of Shikari",
	Long:  `Prints the version of Shikari`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
