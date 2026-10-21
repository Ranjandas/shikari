package shikari

import (
	"path/filepath"
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

func TestTargetCounts(t *testing.T) {
	tests := []struct {
		name                          string
		scale                         bool
		numServers, numClients        uint8
		currentServers, currentClient uint8
		wantServers, wantClients      uint8
	}{
		{"create", false, 3, 2, 0, 0, 3, 2},
		{"create without clients", false, 1, 0, 0, 0, 1, 0},
		{"scale clients only", true, 0, 5, 3, 2, 3, 5},
		{"scale servers only", true, 5, 0, 3, 2, 5, 2},
		{"scale both", true, 1, 1, 3, 2, 1, 1},
	}

	for _, tt := range tests {
		c := ShikariCluster{NumServers: tt.numServers, NumClients: tt.numClients}

		servers, clients := c.targetCounts(tt.scale, tt.currentServers, tt.currentClient)
		if servers != tt.wantServers || clients != tt.wantClients {
			t.Errorf("%s: targetCounts = (%d, %d), want (%d, %d)", tt.name, servers, clients, tt.wantServers, tt.wantClients)
		}
	}
}

func TestTemplateLocator(t *testing.T) {
	abs, _ := filepath.Abs("hashibox.yaml")

	tests := map[string]string{
		"hashibox":                   "template://hashibox",
		"template://hashibox":        "template://hashibox",
		"https://example.com/x.yaml": "https://example.com/x.yaml",
		"hashibox.yaml":              abs,
		"/tmp/Hashibox.YML":          "/tmp/Hashibox.YML",
	}

	for in, want := range tests {
		got, err := templateLocator(in)
		if err != nil || got != want {
			t.Errorf("templateLocator(%q) = %q, %v, want %q", in, got, err, want)
		}
	}
}

func TestInheritFrom(t *testing.T) {
	vm := lima.LimaVM{
		Name:   "dev-srv-01",
		Arch:   "x86_64",
		Config: lima.Config{Env: map[string]string{"SHIKARI_TEMPLATE": "template://hashibox"}},
	}

	c := ShikariCluster{Name: "dev", Arch: "aarch64", Template: "./hashibox.yaml"}
	if w := c.InheritFrom(vm, false, false); w != "" || c.Arch != "x86_64" || c.Template != "template://hashibox" {
		t.Errorf("InheritFrom = %+v, warning %q", c, w)
	}

	c = ShikariCluster{Name: "dev", Arch: "aarch64", Template: "./other.yaml"}
	if w := c.InheritFrom(vm, true, true); w != "" || c.Arch != "aarch64" || c.Template != "./other.yaml" {
		t.Errorf("InheritFrom with flags set = %+v, warning %q", c, w)
	}

	old := lima.LimaVM{Name: "dev-srv-01", Arch: "x86_64"}
	c = ShikariCluster{Name: "dev", Arch: "aarch64", Template: "./hashibox.yaml"}
	if w := c.InheritFrom(old, false, false); w == "" || c.Template != "./hashibox.yaml" || c.Arch != "x86_64" {
		t.Errorf("InheritFrom without recorded template = %+v, warning %q", c, w)
	}
}
