/*
Copyright © 2024 Ranjandas Athiyanathum Poyil thejranjan@gmail.com
*/
package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	lima "github.com/ranjandas/shikari/app/lima"
	"github.com/spf13/cobra"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute commands inside the VMs",
	Long: `Execute commands inside the VMs.

You can run commands against specific class of servers (clients, servers or all)
or a single instance. A single argument is run by a shell inside the VM, so it
can use pipes and variables. Multiple arguments are run as-is, keeping their
quoting. The exit status is the worst exit status of the remote commands.

For example:

$ shikari exec -n murphy -s sudo systemctl is-enabled consul
$ shikari exec -n murphy -a 'cat /etc/hosts | grep srv'
$ shikari exec -n murphy -i srv-01 -- sh -c 'echo $HOSTNAME'`,
	RunE: func(cmd *cobra.Command, args []string) error {

		quiet, _ := cmd.Flags().GetBool("quiet")
		execAll, _ := cmd.Flags().GetBool("all")
		execServers, _ := cmd.Flags().GetBool("servers")
		execClients, _ := cmd.Flags().GetBool("clients")
		execInstance, _ := cmd.Flags().GetString("instance")

		clusterName := cluster.Name

		if len(args) == 0 {
			return fmt.Errorf("no commands provided as args to execute")
		}

		if !execAll && !execServers && !execClients && execInstance == "" {
			return fmt.Errorf("no target given, pass one of -a, -s, -c or -i")
		}

		clusterInstances, err := lima.GetClusterInstances(clusterName)
		if err != nil {
			return err
		}

		instances := lima.GetInstancesByStatus(clusterInstances, "running")

		if len(instances) == 0 {
			return fmt.Errorf("there are no running instances in the cluster %s", clusterName)
		}

		targets, err := execTargets(instances, clusterName, execServers, execClients, execInstance)
		if err != nil {
			return err
		}

		// keep running against the remaining VMs when one fails, and exit
		// with the worst exit status of the remote commands
		var errs []error
		worst := 0
		for _, vm := range targets {
			if err := lima.ExecLimaVM(vm.Name, args, quiet); err != nil {
				errs = append(errs, err)

				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) && exitErr.ExitCode() > worst {
					worst = exitErr.ExitCode()
				}
			}
		}

		if len(errs) == 0 {
			return nil
		}

		return &exitCodeError{code: worst, err: errors.Join(errs...)}
	},
}

var instanceSelectorPattern = regexp.MustCompile(`^(srv|cli)-(\d+)$`)

// execTargets returns the instances selected by the -s, -c or -i flags, or all
// instances when none of them is given (-a).
func execTargets(instances []lima.LimaVM, clusterName string, servers bool, clients bool, instance string) ([]lima.LimaVM, error) {
	switch {
	case servers:
		return lima.GetServerInstances(instances), nil
	case clients:
		return lima.GetClientInstances(instances), nil
	case instance != "":
		// accept srv-01, srv-1 or murphy-srv-01, but never a partial match
		match := instanceSelectorPattern.FindStringSubmatch(strings.TrimPrefix(instance, clusterName+"-"))
		if match == nil {
			return nil, fmt.Errorf("invalid instance name %q, expected eg: srv-01 or cli-02", instance)
		}

		index, _ := strconv.Atoi(match[2])
		for _, vm := range instances {
			if vm.Index() == index && ((match[1] == "srv" && vm.IsServer()) || (match[1] == "cli" && vm.IsClient())) {
				return []lima.LimaVM{vm}, nil
			}
		}

		return nil, fmt.Errorf("no running instance %s in cluster %s", instance, clusterName)
	default:
		return instances, nil
	}
}

func init() {
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().BoolP("quiet", "q", false, "prints only the output of the commands without headers")
	execCmd.Flags().BoolP("clients", "c", false, "run commands against client instances in the cluster")
	execCmd.Flags().BoolP("servers", "s", false, "run commands against server instances in the cluster")
	execCmd.Flags().BoolP("all", "a", false, "run commands against all instances in the cluster")
	execCmd.Flags().StringP("instance", "i", "", "name of the specific instance to run the command against")
	addNameFlag(execCmd, true)

	execCmd.MarkFlagsMutuallyExclusive("clients", "servers", "all", "instance")

}
