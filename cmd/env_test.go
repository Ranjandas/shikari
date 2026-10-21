package cmd

import (
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

func TestRandomServerAddressIsPickedOnce(t *testing.T) {
	c := ClientConfigOpts{Name: "murphy", serverAddr: "192.168.105.2"}

	// no instances are needed, as the address is already picked
	addr, err := c.getRandomServerAddress()
	if err != nil || addr != "192.168.105.2" {
		t.Errorf("getRandomServerAddress = %q, %v, want the picked address", addr, err)
	}
}
