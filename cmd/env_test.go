package cmd

import (
	"strings"
	"testing"

	lima "github.com/ranjandas/shikari/app/lima"
)

func TestFirstServer(t *testing.T) {
	c := ClientConfigOpts{Name: "murphy", instances: []lima.LimaVM{
		{Name: "murphy-cli-01"},
		{Name: "murphy-srv-01", Dir: "/lima/murphy-srv-01"},
	}}

	vm, ok := c.firstServer()
	if !ok || vm.Dir != "/lima/murphy-srv-01" {
		t.Errorf("firstServer = %+v, %v, want murphy-srv-01", vm, ok)
	}

	if got := c.getTLSCaCertPath("consul"); got != "/lima/murphy-srv-01/copied-from-guest/consul-agent-ca.pem" {
		t.Errorf("getTLSCaCertPath = %q", got)
	}

	c.instances = c.instances[:1]
	if _, ok := c.firstServer(); ok {
		t.Error("firstServer found a server in a cluster without one")
	}
}

func TestServerAddressIsPickedOnce(t *testing.T) {
	c := ClientConfigOpts{Name: "murphy", serverAddr: "192.168.105.2"}

	// no instances are needed, as the address is already picked
	addr, err := c.getServerAddress()
	if err != nil || addr != "192.168.105.2" {
		t.Errorf("getServerAddress = %q, %v, want the picked address", addr, err)
	}
}

func TestPickServer(t *testing.T) {
	instances := []lima.LimaVM{
		{Name: "murphy-cli-01", Status: "Running"},
		{Name: "murphy-srv-03", Status: "Running"},
		{Name: "murphy-srv-01", Status: "Stopped"},
		{Name: "murphy-srv-02", Status: "Running"},
	}

	tests := []struct {
		server  string
		want    string
		wantErr string
	}{
		{"", "murphy-srv-02", ""},
		{"srv-03", "murphy-srv-03", ""},
		{"murphy-srv-03", "murphy-srv-03", ""},
		{"srv-01", "", "server murphy-srv-01 is stopped"},
		{"srv-09", "", "server murphy-srv-09 not found"},
		{"cli-01", "", "server murphy-cli-01 not found"},
	}

	for _, tt := range tests {
		vm, err := pickServer(instances, "murphy", tt.server)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("pickServer(%q) error = %v, want %q", tt.server, err, tt.wantErr)
			}
			continue
		}
		if err != nil || vm.Name != tt.want {
			t.Errorf("pickServer(%q) = %s, %v, want %s", tt.server, vm.Name, err, tt.want)
		}
	}
}

func TestPickServerNoneRunning(t *testing.T) {
	_, err := pickServer([]lima.LimaVM{{Name: "murphy-srv-01", Status: "Stopped"}}, "murphy", "")
	if err == nil || !strings.Contains(err.Error(), "murphy-srv-01 (stopped)") {
		t.Errorf("pickServer error = %v, want the stopped servers listed", err)
	}

	_, err = pickServer([]lima.LimaVM{{Name: "murphy-cli-01", Status: "Running"}}, "murphy", "")
	if err == nil || !strings.Contains(err.Error(), "has no servers") {
		t.Errorf("pickServer error = %v, want no servers", err)
	}
}
