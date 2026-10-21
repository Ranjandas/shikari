package lima

import (
	"os"
	"path/filepath"
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

func TestPickIPAddress(t *testing.T) {
	iface := func(name string, addrs ...string) Interface {
		i := Interface{IfName: name}
		for _, addr := range addrs {
			i.AddrInfo = append(i.AddrInfo, AddrInfo{Family: "inet", Local: addr})
		}
		return i
	}

	tests := []struct {
		name       string
		interfaces []Interface
		want       string
	}{
		{"lima0 wins", []Interface{iface("lo", "127.0.0.1"), iface("eth0", "192.168.5.15"), iface("docker0", "172.17.0.1"), iface("lima0", "192.168.105.4")}, "192.168.105.4"},
		{"other lima interface", []Interface{iface("eth0", "192.168.5.15"), iface("lima1", "192.168.106.2")}, "192.168.106.2"},
		{"vzNAT on eth0", []Interface{iface("lo", "127.0.0.1"), iface("cni0", "10.42.0.1"), iface("eth0", "192.168.64.7")}, "192.168.64.7"},
		{"user-mode only", []Interface{iface("lo", "127.0.0.1"), iface("eth0", "192.168.5.15")}, ""},
		{"no interfaces", nil, ""},
	}

	for _, tt := range tests {
		if got := pickIPAddress(tt.interfaces); got != tt.want {
			t.Errorf("%s: pickIPAddress = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestGetIPAddresses(t *testing.T) {
	// a fake limactl that answers `limactl shell <vm> ip -j -4 addr show`
	dir := t.TempDir()
	script := `#!/bin/sh
case "$2" in
murphy-srv-01) echo '[{"ifname":"lima0","addr_info":[{"family":"inet","local":"192.168.105.2"}]}]' ;;
murphy-cli-01) echo '[{"ifname":"lima0","addr_info":[{"family":"inet","local":"192.168.105.3"}]}]' ;;
*) echo 'not json' ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "limactl"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got := GetIPAddresses([]LimaVM{
		{Name: "murphy-srv-01", Status: "Running"},
		{Name: "murphy-cli-01", Status: "Running"},
		{Name: "murphy-cli-02", Status: "Running"},
		{Name: "murphy-cli-03", Status: "Stopped"},
	})

	want := map[string]string{
		"murphy-srv-01": "192.168.105.2",
		"murphy-cli-01": "192.168.105.3",
		"murphy-cli-02": "",
		"murphy-cli-03": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetIPAddresses = %v, want %v", got, want)
	}
}
