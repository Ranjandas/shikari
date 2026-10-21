package shikari

import (
	"testing"

	lima "github.com/ranjandas/shikari/app/lima"
)

func TestCountVMs(t *testing.T) {
	vms := []lima.LimaVM{
		{Name: "dev-srv-01"},
		{Name: "dev-srv-02"},
		{Name: "dev-cli-01"},
		// not a server or client, must not be counted as a server
		{Name: "dev-postgres"},
	}

	servers, clients := countVMs(vms)
	if servers != 2 || clients != 1 {
		t.Errorf("countVMs = (%d, %d), want (2, 1)", servers, clients)
	}
}
