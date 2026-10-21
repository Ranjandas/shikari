package lima

import (
	"reflect"
	"testing"
)

func TestParseInstances(t *testing.T) {
	output := []byte(`{"name":"dev-srv-01","status":"Running","config":{"env":{"SHIKARI_CLUSTER_NAME":"dev"}}}
{"name":"default","status":"Stopped"}
`)

	vms, err := parseInstances(output)
	if err != nil {
		t.Fatal(err)
	}

	if len(vms) != 2 || vms[0].Name != "dev-srv-01" || vms[1].Status != "Stopped" {
		t.Errorf("parseInstances = %+v", vms)
	}

	if vms[0].ClusterName() != "dev" {
		t.Errorf("ClusterName = %q, want dev", vms[0].ClusterName())
	}

	if vms, err := parseInstances(nil); err != nil || len(vms) != 0 {
		t.Errorf("parseInstances(nil) = %v, %v", vms, err)
	}

	if _, err := parseInstances([]byte("not json\n")); err == nil {
		t.Error("parseInstances did not fail on invalid output")
	}
}

func TestExecArgs(t *testing.T) {
	got := execArgs("m-srv-01", []string{"cat /etc/hosts | grep srv"})
	want := []string{"shell", "m-srv-01", "sh", "-c", "cat /etc/hosts | grep srv"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("execArgs(single) = %q, want %q", got, want)
	}

	got = execArgs("m-srv-01", []string{"sh", "-c", "echo $HOSTNAME"})
	want = []string{"shell", "m-srv-01", "sh", "-c", "echo $HOSTNAME"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("execArgs(multiple) = %q, want %q", got, want)
	}
}
