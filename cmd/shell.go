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
	Run: func(cmd *cobra.Command, args []string) {
		if !(len(args) > 0) {
			fmt.Println("No Instance name passed")
			return
		}

		vm := lima.GetInstance(args[0])
		// return if no instance with the name was found
		if vm.Name == "" || vm.Status != "Running" {
			fmt.Printf("No running instance found with the name \"%s\"\n", args[0])
			return
		}

		lima.ShellLimaVM(vm.Name)

	},
}

func init() {
	rootCmd.AddCommand(shellCmd)

	usageString := "Usage:\n shikari shell <vm-name> [flags]\n\nFlags:\n -h, --help help for shell\n"

	shellCmd.SetUsageTemplate(usageString)
}
