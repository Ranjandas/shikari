package cmd

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ranjandas/shikari/internal/limatest"
)

func TestListJSON(t *testing.T) {
	limatest.Install(t,
		`{"name":"murphy-srv-01","status":"Stopped","arch":"aarch64","cpus":4,"memory":4294967296,"disk":107374182400,"config":{"env":{"SHIKARI_CLUSTER_NAME":"murphy","SHIKARI_SCENARIO_NAME":"nomad"},"images":[{"location":"/images/c.qcow2","arch":"aarch64"}]}}`,
		`{"name":"murphy-cli-01","status":"Stopped","arch":"aarch64","config":{"env":{"SHIKARI_CLUSTER_NAME":"murphy"}}}`,
		`{"name":"other-srv-01","status":"Stopped","arch":"aarch64","config":{"env":{"SHIKARI_CLUSTER_NAME":"other"}}}`,
		`{"name":"default","status":"Stopped","arch":"aarch64"}`,
	)

	var out bytes.Buffer
	if err := listInstances(&out, "murphy", "json"); err != nil {
		t.Fatalf("listInstances = %v", err)
	}

	var got []listedVM
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}

	want := []listedVM{
		{Cluster: "murphy", Name: "murphy-srv-01", Role: "server", Arch: "aarch64", Status: "Stopped", Scenario: "nomad", DiskBytes: 107374182400, MemoryBytes: 4294967296, CPUs: 4, Image: "/images/c.qcow2"},
		{Cluster: "murphy", Name: "murphy-cli-01", Role: "client", Arch: "aarch64", Status: "Stopped"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("list -o json = %+v, want %+v", got, want)
	}

	for _, key := range []string{`"cluster"`, `"role"`, `"ip"`, `"disk_bytes"`, `"memory_bytes"`} {
		if !strings.Contains(out.String(), key) {
			t.Errorf("output has no %s key:\n%s", key, out.String())
		}
	}
}

func TestListJSONEmpty(t *testing.T) {
	limatest.Install(t)

	var out bytes.Buffer
	if err := listInstances(&out, "murphy", "json"); err != nil {
		t.Fatalf("listInstances = %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "[]" {
		t.Errorf("list -o json with no VMs = %q, want []", got)
	}
}

func TestListRejectsUnknownOutput(t *testing.T) {
	var out bytes.Buffer
	if err := listInstances(&out, "", "yaml"); err == nil || !strings.Contains(err.Error(), "yaml") {
		t.Errorf("listInstances with -o yaml = %v, want an error naming it", err)
	}
}
