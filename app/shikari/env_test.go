package shikari

import (
	"strings"
	"testing"
)

func TestGenerateEnvArgsReturnsErrors(t *testing.T) {
	for _, env := range []string{"NOVALUE", "EMPTY=", "=secret", "BAD KEY=secret"} {
		c := ShikariCluster{EnvVars: []string{env}}

		err := c.generateEnvArgs(&yqSetter{})
		if err == nil {
			t.Errorf("generateEnvArgs(%q) did not fail", env)
			continue
		}

		if strings.Contains(err.Error(), "secret") {
			t.Errorf("generateEnvArgs(%q) error leaks the value: %v", env, err)
		}
	}

	c := ShikariCluster{EnvVars: []string{"CONSUL_LICENSE=abc=def"}}
	s := yqSetter{}
	if err := c.generateEnvArgs(&s); err != nil {
		t.Fatal(err)
	}
	if got := assignments(t, s.expression())[".env.CONSUL_LICENSE"]; got != "abc=def" {
		t.Errorf("CONSUL_LICENSE = %q, want abc=def", got)
	}
}

func TestCreateClusterRejectsInvalidName(t *testing.T) {
	c := ShikariCluster{Name: "my-cluster"}

	if err := c.CreateCluster(false); err == nil {
		t.Error("CreateCluster did not fail for an invalid name")
	}
}
