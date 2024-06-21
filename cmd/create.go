/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates multiple VMs to form a cluster.",
	Long: `Creates multiple VMs to form a cluster

For example:

$ shikari create --name murphy --servers 3  --clients 3 --template hashibox --env CONSUL_LICENSE=$(cat consul.hclic)

The above command will create a 3 server and 3 client cluster, each vm
carrying the name as a prefix to easily identify.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(lima.GetInstancesByPrefix(cluster.Name)) > 0 {
			fmt.Printf("Cluster %s alredy exist!", cluster.Name)
			return
		}
		cluster.CreateCluster(false)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	createCmd.Flags().Uint8VarP(&cluster.NumServers, "servers", "s", 1, "number of servers")
	createCmd.Flags().Uint8VarP(&cluster.NumClients, "clients", "c", 0, "number of clients")
	createCmd.Flags().StringVarP(&cluster.Name, "name", "n", "", "name of the cluster")
	createCmd.Flags().StringVarP(&cluster.Template, "template", "t", "./hashibox.yaml", "name of lima template for the VMs")
	createCmd.Flags().StringSliceVarP(&cluster.EnvVars, "env", "e", []string{}, "provide environment vars in the for key=value (can be used multiple times)")
	createCmd.Flags().StringVarP(&cluster.ImgPath, "image", "i", "", "path to the cqow2 images to be used for the VMs, overriding the one in the template")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("servers")
}
