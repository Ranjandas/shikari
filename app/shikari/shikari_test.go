package shikari

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	lima "github.com/ranjandas/shikari/app/lima"
)

func vms(names ...string) []lima.LimaVM {
	var v []lima.LimaVM
	for _, n := range names {
		v = append(v, lima.LimaVM{Name: n})
	}
	return v
}

func TestPlanRole(t *testing.T) {
	c := ShikariCluster{Name: "dev"}

	tests := []struct {
		name        string
		existing    []lima.LimaVM
		target      uint8
		wantCreate  []string
		wantDestroy []string
	}{
		{"scale up", vms("dev-cli-01", "dev-cli-02"), 4, []string{"dev-cli-03", "dev-cli-04"}, nil},
		{"fill gap", vms("dev-cli-01", "dev-cli-03"), 3, []string{"dev-cli-02"}, nil},
		{"fill gap and grow", vms("dev-cli-02"), 3, []string{"dev-cli-01", "dev-cli-03"}, nil},
		{"scale down removes highest", vms("dev-cli-03", "dev-cli-01", "dev-cli-04"), 1, nil, []string{"dev-cli-04", "dev-cli-03"}},
		{"scale to zero", vms("dev-cli-01", "dev-cli-02"), 0, nil, []string{"dev-cli-02", "dev-cli-01"}},
		{"unchanged", vms("dev-cli-01"), 1, nil, nil},
		{"from nothing", nil, 2, []string{"dev-cli-01", "dev-cli-02"}, nil},
	}

	for _, tt := range tests {
		create, destroy := c.planRole("cli", tt.existing, tt.target)
		if !reflect.DeepEqual(create, tt.wantCreate) || !reflect.DeepEqual(destroy, tt.wantDestroy) {
			t.Errorf("%s: planRole = (%v, %v), want (%v, %v)", tt.name, create, destroy, tt.wantCreate, tt.wantDestroy)
		}
	}
}

func TestTargetCounts(t *testing.T) {
	tests := []struct {
		name                           string
		scale                          bool
		numServers, numClients         uint8
		serversSet, clientsSet         bool
		currentServers, currentClients uint8
		wantServers, wantClients       uint8
	}{
		{"create", false, 3, 2, true, true, 0, 0, 3, 2},
		{"create without clients", false, 1, 0, true, false, 0, 0, 1, 0},
		{"scale clients only", true, 0, 5, false, true, 3, 2, 3, 5},
		{"scale servers only", true, 5, 0, true, false, 3, 2, 5, 2},
		{"scale both", true, 1, 1, true, true, 3, 2, 1, 1},
		{"scale clients to zero", true, 0, 0, false, true, 3, 2, 3, 0},
	}

	for _, tt := range tests {
		c := ShikariCluster{NumServers: tt.numServers, NumClients: tt.numClients, ServersSet: tt.serversSet, ClientsSet: tt.clientsSet}

		servers, clients := c.targetCounts(tt.scale, tt.currentServers, tt.currentClients)
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

func TestCheckImage(t *testing.T) {
	dir := t.TempDir()
	image := filepath.Join(dir, "disk.raw")
	if err := os.WriteFile(image, []byte("raw disk"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := checkImage(image); err != nil {
		t.Errorf("checkImage(raw image) = %v", err)
	}

	err := checkImage(filepath.Join(dir, "typo.qcow2"))
	if err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("checkImage(missing) = %v, want a no such file error", err)
	}

	if err := checkImage(dir); err == nil {
		t.Error("checkImage(directory) did not fail")
	}
}
