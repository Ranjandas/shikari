/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"

	"github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Get a shell inside the VM",
	Long:  `Get a shell inside the VM`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !(len(args) > 0) {
			return fmt.Errorf("no instance name passed")
		}

		vm, err := lima.GetInstance(args[0])
		if err != nil {
			return err
		}

		// return if no instance with the name was found
		if vm.Name == "" || vm.Status != "Running" {
			return fmt.Errorf("no running instance found with the name \"%s\"", args[0])
		}

		return lima.ShellLimaVM(vm.Name)
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)

	usageString := "Usage:\n shikari shell <vm-name>\n\nFlags:\n -h, --help help for shell\n"

	shellCmd.SetUsageTemplate(usageString)
}
