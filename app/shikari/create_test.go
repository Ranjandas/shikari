package shikari

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ranjandas/shikari/internal/limatest"
)

// instance is a `limactl list --json` line for a VM of the murphy cluster.
func instance(name string, status string) string {
	return fmt.Sprintf(`{"name":%q,"status":%q,"arch":"aarch64","config":{"env":{"SHIKARI_CLUSTER_NAME":"murphy"}}}`, name, status)
}

// startedVMs returns the names of the VMs `limactl start --name` was called for.
func startedVMs(calls [][]string) []string {
	var names []string
	for _, call := range calls {
		if len(call) > 2 && call[0] == "start" && call[1] == "--name" {
			names = append(names, call[2])
		}
	}
	return names
}

// injectedEnv returns the .env values the --set expression of vmName sets.
func injectedEnv(t *testing.T, calls [][]string, vmName string) map[string]string {
	t.Helper()

	var expression string
	for _, call := range calls {
		if len(call) > 2 && call[0] == "start" && call[2] == vmName {
			for i, arg := range call {
				if arg == "--set" && i+1 < len(call) {
					expression = call[i+1]
				}
			}
		}
	}

	injected := map[string]string{}
	for path, value := range assignments(t, expression) {
		if key, ok := strings.CutPrefix(path, ".env."); ok {
			injected[key] = value
		}
	}

	return injected
}

func TestCreateClusterSpawnsEachVM(t *testing.T) {
	fake := limatest.Install(t)

	c := ShikariCluster{Name: "murphy", Arch: "aarch64", NumServers: 2, NumClients: 1, Template: "hashibox", EnvVars: []string{"CONSUL_LICENSE=a 'quoted' license"}}
	if err := c.CreateCluster(false); err != nil {
		t.Fatalf("CreateCluster = %v", err)
	}

	calls := fake.Calls()
	if got, want := startedVMs(calls), []string{"murphy-cli-01", "murphy-srv-01", "murphy-srv-02"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("started %v, want %v", got, want)
	}

	for _, call := range calls {
		joined := strings.Join(call, " ")
		if !strings.Contains(joined, "template://hashibox --arch aarch64 --tty=false --set") {
			t.Errorf("limactl %s: want the template and arch", joined)
		}
		if strings.Contains(joined, "quoted") || strings.Contains(joined, "env(") {
			t.Errorf("limactl %s: want the values base64 encoded, without env operations", joined)
		}
	}

	env := injectedEnv(t, calls, "murphy-cli-01")
	want := map[string]string{
		"SHIKARI_CLUSTER_NAME":  "murphy",
		"SHIKARI_TEMPLATE":      "template://hashibox",
		"SHIKARI_SCENARIO_NAME": "hashibox",
		"SHIKARI_SERVER_COUNT":  "2",
		"SHIKARI_CLIENT_COUNT":  "1",
		"SHIKARI_LAUNCH_MODE":   "create",
		"SHIKARI_VM_MODE":       "client",
		"CONSUL_LICENSE":        "a 'quoted' license",
	}
	if !reflect.DeepEqual(env, want) {
		t.Errorf("murphy-cli-01 env = %v, want %v", env, want)
	}

	if mode := injectedEnv(t, calls, "murphy-srv-02")["SHIKARI_VM_MODE"]; mode != "server" {
		t.Errorf("murphy-srv-02 SHIKARI_VM_MODE = %q, want server", mode)
	}
}

func TestCreateClusterReportsFailedVMs(t *testing.T) {
	fake := limatest.Install(t)
	fake.FailOn("murphy-srv-02")

	c := ShikariCluster{Name: "murphy", Arch: "aarch64", NumServers: 2, Template: "hashibox"}
	err := c.CreateCluster(false)
	if err == nil || !strings.Contains(err.Error(), "murphy-srv-02") {
		t.Fatalf("CreateCluster = %v, want the failed VM", err)
	}
	if strings.Contains(err.Error(), "murphy-srv-01") {
		t.Errorf("CreateCluster = %v, murphy-srv-01 did not fail", err)
	}
}

func TestScaleUpFillsGaps(t *testing.T) {
	fake := limatest.Install(t,
		instance("murphy-srv-01", "Running"),
		instance("murphy-srv-03", "Running"),
		instance("murphy-cli-01", "Running"),
	)

	c := ShikariCluster{Name: "murphy", Arch: "aarch64", NumServers: 3, ServersSet: true, Template: "hashibox"}
	if err := c.CreateCluster(true); err != nil {
		t.Fatalf("CreateCluster(scale) = %v", err)
	}

	calls := fake.Calls()
	if got, want := startedVMs(calls), []string{"murphy-srv-02"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("started %v, want %v", got, want)
	}

	env := injectedEnv(t, calls, "murphy-srv-02")
	if env["SHIKARI_SERVER_COUNT"] != "3" || env["SHIKARI_CLIENT_COUNT"] != "1" || env["SHIKARI_LAUNCH_MODE"] != "scale" {
		t.Errorf("murphy-srv-02 env = %v, want 3 servers, 1 client, scale", env)
	}
}

func TestScaleDown(t *testing.T) {
	existing := []string{
		instance("murphy-srv-01", "Running"),
		instance("murphy-srv-02", "Running"),
		instance("murphy-srv-03", "Running"),
	}

	t.Run("refused without force", func(t *testing.T) {
		fake := limatest.Install(t, existing...)

		c := ShikariCluster{Name: "murphy", NumServers: 1, ServersSet: true, Template: "hashibox"}
		err := c.CreateCluster(true)
		if err == nil || !strings.Contains(err.Error(), "murphy-srv-03,murphy-srv-02") {
			t.Errorf("CreateCluster(scale) = %v, want the VMs that would be destroyed", err)
		}
		if calls := fake.Calls(); len(calls) != 0 {
			t.Errorf("limactl was called: %v", calls)
		}
	})

	t.Run("forced", func(t *testing.T) {
		fake := limatest.Install(t, existing...)

		c := ShikariCluster{Name: "murphy", NumServers: 1, ServersSet: true, Template: "hashibox", Force: true}
		if err := c.CreateCluster(true); err != nil {
			t.Fatalf("CreateCluster(scale) = %v", err)
		}

		want := [][]string{{"delete", "-f", "murphy-srv-02"}, {"delete", "-f", "murphy-srv-03"}}
		if got := fake.Calls(); !reflect.DeepEqual(got, want) {
			t.Errorf("limactl calls = %v, want %v", got, want)
		}
	})
}
