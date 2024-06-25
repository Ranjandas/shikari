/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"fmt"

	shikari "github.com/ranjandas/shikari/app/shikari"
	"github.com/spf13/cobra"
)

// scaleCmd represents the scale command
var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale the number of VMs in the cluster",
	Long:  `Scale the number of VMs in the cluster`,
	Run: func(cmd *cobra.Command, args []string) {

		if cluster.NumServers > 5 {
			fmt.Println("Servers are not recommended to be more than 5!")
			return
		}

		cluster.CreateCluster(true)
	},
}

var cluster shikari.ShikariCluster

func init() {
	rootCmd.AddCommand(scaleCmd)

	scaleCmd.Flags().Uint8VarP(&cluster.NumServers, "servers", "s", 0, "number of servers")
	scaleCmd.Flags().Uint8VarP(&cluster.NumClients, "clients", "c", 0, "number of clients")
	scaleCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	scaleCmd.Flags().StringVarP(&cluster.Template, "template", "t", "./hashibox.yaml", "name of lima template for the VMs")
	scaleCmd.Flags().StringSliceVarP(&cluster.EnvVars, "env", "e", []string{}, "provide environment vars in the for key=value (can be used multiple times)")
	scaleCmd.Flags().StringVarP(&cluster.ImgPath, "image", "i", "", "path to the cqow2 images to be used for the VMs, overriding the one in the template")
	scaleCmd.Flags().BoolVarP(&cluster.Force, "force", "f", false, "force scaling down of the cluster VMs")
	scaleCmd.MarkFlagRequired("name")
	scaleCmd.MarkFlagRequired("clients")
}
