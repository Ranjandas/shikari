package lima

import (
	"reflect"
	"testing"
)

func vm(name, clusterEnv string) LimaVM {
	v := LimaVM{Name: name}
	if clusterEnv != "" {
		v.Config.Env = map[string]string{"SHIKARI_CLUSTER_NAME": clusterEnv}
	}
	return v
}

func names(vms []LimaVM) []string {
	var n []string
	for _, v := range vms {
		n = append(n, v.Name)
	}
	return n
}

func TestClusterName(t *testing.T) {
	tests := []struct {
		vm   LimaVM
		want string
	}{
		{vm("dev-srv-01", "dev"), "dev"},
		{vm("dev-cli-12", "dev"), "dev"},
		// hand-made VM sharing the cluster prefix
		{vm("dev-postgres", ""), ""},
		{vm("dev-postgres", "dev"), ""},
		// right name, but not created by Shikari
		{vm("dev-srv-01", ""), ""},
		// env names a different cluster
		{vm("dev-srv-01", "prod"), ""},
		{vm("devx-srv-01", "devx"), "devx"},
		{vm("default", ""), ""},
	}

	for _, tt := range tests {
		if got := tt.vm.ClusterName(); got != tt.want {
			t.Errorf("ClusterName(%q, env=%v) = %q, want %q", tt.vm.Name, tt.vm.Config.Env, got, tt.want)
		}
	}
}

func TestFilterClusterInstances(t *testing.T) {
	instances := []LimaVM{
		vm("dev-srv-01", "dev"),
		vm("dev-srv-02", "dev"),
		vm("dev-cli-01", "dev"),
		vm("dev-postgres", ""),
		vm("dev-srv-03", ""),
		vm("devx-srv-01", "devx"),
		vm("default", ""),
	}

	got := names(FilterClusterInstances(instances, "dev"))
	want := []string{"dev-srv-01", "dev-srv-02", "dev-cli-01"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterClusterInstances(dev) = %v, want %v", got, want)
	}

	if got := FilterClusterInstances(instances, ""); len(got) != 0 {
		t.Errorf("FilterClusterInstances(\"\") = %v, want none", names(got))
	}

	got = names(FilterShikariInstances(instances))
	want = []string{"dev-srv-01", "dev-srv-02", "dev-cli-01", "devx-srv-01"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterShikariInstances = %v, want %v", got, want)
	}

	got = names(GetServerInstances(FilterClusterInstances(instances, "dev")))
	want = []string{"dev-srv-01", "dev-srv-02"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetServerInstances = %v, want %v", got, want)
	}
}

func TestIndex(t *testing.T) {
	tests := map[string]int{"dev-srv-01": 1, "dev-cli-12": 12, "dev-postgres": 0, "dev-srv-1": 1}

	for name, want := range tests {
		if got := (LimaVM{Name: name}).Index(); got != want {
			t.Errorf("Index(%q) = %d, want %d", name, got, want)
		}
	}
}
