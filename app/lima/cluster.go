package lima

import (
	"regexp"
)

// clusterVMPattern matches the names Shikari gives cluster VMs, eg: murphy-srv-01, murphy-cli-02
var clusterVMPattern = regexp.MustCompile(`^([a-zA-Z0-9]+)-(srv|cli)-(\d+)$`)

// ClusterName returns the name of the Shikari cluster the VM belongs to, or an
// empty string if the VM is not part of a Shikari cluster.
//
// A VM belongs to a cluster only when its name follows the Shikari naming
// scheme and the SHIKARI_CLUSTER_NAME env var Shikari writes into the Lima
// config names the same cluster. Matching on the name prefix alone would pick
// up unrelated Lima VMs such as "dev-postgres" for the cluster "dev".
func (vm LimaVM) ClusterName() string {
	match := clusterVMPattern.FindStringSubmatch(vm.Name)
	if match == nil {
		return ""
	}

	if vm.Config.Env["SHIKARI_CLUSTER_NAME"] != match[1] {
		return ""
	}

	return match[1]
}

// IsClient reports whether the VM is a client (-cli-) VM of a Shikari cluster.
func (vm LimaVM) IsClient() bool {
	match := clusterVMPattern.FindStringSubmatch(vm.Name)
	return match != nil && match[2] == "cli"
}

// IsServer reports whether the VM is a server (-srv-) VM of a Shikari cluster.
func (vm LimaVM) IsServer() bool {
	match := clusterVMPattern.FindStringSubmatch(vm.Name)
	return match != nil && match[2] == "srv"
}

// FilterClusterInstances returns the instances that belong to the named cluster.
func FilterClusterInstances(instances []LimaVM, clusterName string) []LimaVM {
	var filteredInstances []LimaVM

	if clusterName == "" {
		return filteredInstances
	}

	for _, instance := range instances {
		if instance.ClusterName() == clusterName {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	return filteredInstances
}

// FilterShikariInstances returns the instances that belong to any Shikari cluster.
func FilterShikariInstances(instances []LimaVM) []LimaVM {
	var filteredInstances []LimaVM

	for _, instance := range instances {
		if instance.ClusterName() != "" {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	return filteredInstances
}

// GetClusterInstances returns the Lima VMs that belong to the named cluster.
func GetClusterInstances(clusterName string) ([]LimaVM, error) {
	instances, err := ListInstances()
	if err != nil {
		return nil, err
	}

	return FilterClusterInstances(instances, clusterName), nil
}

// GetServerInstances returns only the server VMs from the given instances.
func GetServerInstances(instances []LimaVM) []LimaVM {
	var filteredInstances []LimaVM

	for _, instance := range instances {
		if instance.IsServer() {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	return filteredInstances
}
