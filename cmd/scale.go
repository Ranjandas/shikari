/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"
	"os"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// scaleCmd represents the scale command
var scaleCmd = &cobra.Command{
	Use:     "scale",
	Short:   "Scale the number of VMs in the cluster",
	Long:    `Scale the number of VMs in the cluster`,
	PreRunE: loadLicenses,
	RunE: func(cmd *cobra.Command, args []string) error {
		instances, err := lima.GetClusterInstances(cluster.Name)
		if err != nil {
			return err
		}

		if len(instances) == 0 {
			return fmt.Errorf("cluster %s not found", cluster.Name)
		}

		cluster.ServersSet = cmd.Flags().Changed("servers")
		cluster.ClientsSet = cmd.Flags().Changed("clients")

		warning := cluster.InheritFrom(instances[0], cmd.Flags().Changed("arch"), cmd.Flags().Changed("template"))
		if warning != "" {
			fmt.Fprintln(os.Stderr, "Warning:", warning)
		}

		return cluster.CreateCluster(true)
	},
}

func init() {
	rootCmd.AddCommand(scaleCmd)

	scaleCmd.Flags().Uint8VarP(&cluster.NumServers, "servers", "s", 0, "number of servers")
	scaleCmd.Flags().Uint8VarP(&cluster.NumClients, "clients", "c", 0, "number of clients")
	scaleCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	scaleCmd.Flags().StringVarP(&cluster.Template, "template", "t", "./hashibox.yaml", "name of lima template for the VMs (defaults to the template the cluster was created with)")
	scaleCmd.Flags().StringSliceVarP(&cluster.EnvVars, "env", "e", []string{}, "provide environment vars in the for key=value (can be used multiple times)")
	scaleCmd.Flags().StringVarP(&cluster.ImgPath, "image", "i", "", "path to the disk image (qcow2 or raw) to be used for the VMs, overriding the one in the template")
	scaleCmd.Flags().BoolVarP(&cluster.Force, "force", "f", false, "force scaling down of the cluster VMs")
	scaleCmd.Flags().StringVarP(&cluster.Arch, "arch", "a", defaultArch(), "the architecture of the VM (supported by Lima), defaults to the arch of the cluster. Eg: aarch64, x86_64")

	scaleCmd.MarkFlagRequired("name")
	scaleCmd.MarkFlagsOneRequired("clients", "servers")
}
