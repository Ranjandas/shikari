package shikari

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

var assignment = regexp.MustCompile(`(\.[\w.\[\]]+) = (?:\(\.[\w.]+ // )?\("([A-Za-z0-9+/=]*)" \| @base64d\)\)?`)

// assignments resolves the `.x = ("..." | @base64d)` and
// `.x = (.x // ("..." | @base64d))` parts of a --set expression, by path.
func assignments(t *testing.T, expression string) map[string]string {
	t.Helper()

	values := map[string]string{}
	for _, match := range assignment.FindAllStringSubmatch(expression, -1) {
		value, err := base64.StdEncoding.DecodeString(match[2])
		if err != nil {
			t.Fatalf("%s: %v", match[0], err)
		}
		values[match[1]] = string(value)
	}

	return values
}

func TestYQSetterKeepsValuesOutOfExpression(t *testing.T) {
	values := []string{
		`x'; touch /tmp/pwned; '`,
		`say "hi" | .env.EVIL = "1"`,
		"multi\nline\\value",
	}

	s := yqSetter{}
	for _, v := range values {
		if err := s.setEnv("FOO", v); err != nil {
			t.Fatal(err)
		}
	}

	expr := s.expression()
	for _, v := range values {
		if strings.Contains(expr, v) {
			t.Errorf("expression %q contains value %q", expr, v)
		}
	}

	// Lima v2.1.0 and later fail the expression if it reads the environment
	if strings.Contains(expr, "env(") {
		t.Errorf("expression %q uses env operations", expr)
	}

	if len(s.exprs) != len(values) {
		t.Fatalf("expressions = %q, want %d", s.exprs, len(values))
	}
	for i, v := range values {
		if got := assignments(t, s.exprs[i])[".env.FOO"]; got != v {
			t.Errorf("expression %q sets %q, want %q", s.exprs[i], got, v)
		}
	}
}

func TestYQSetterExpression(t *testing.T) {
	s := yqSetter{}
	s.setEnv("SHIKARI_CLUSTER_NAME", "murphy")
	s.set(".images[0].location", "/a b")

	want := `.env.SHIKARI_CLUSTER_NAME = ("bXVycGh5" | @base64d) | .images[0].location = ("L2EgYg==" | @base64d)`
	if expr := s.expression(); expr != want {
		t.Errorf("expression = %q, want %q", expr, want)
	}
}

func TestYQSetterRejectsInvalidEnvKeys(t *testing.T) {
	for _, key := range []string{"", "1FOO", "FOO BAR", `FOO"`, "FOO=1", "a.b", "x | .y"} {
		s := yqSetter{}
		if err := s.setEnv(key, "v"); err == nil {
			t.Errorf("setEnv(%q) did not fail", key)
		}
		if len(s.exprs) != 0 {
			t.Errorf("setEnv(%q) changed the setter", key)
		}
	}

	s := yqSetter{}
	for _, key := range []string{"FOO", "_FOO", "CONSUL_LICENSE", "foo1"} {
		if err := s.setEnv(key, "v"); err != nil {
			t.Errorf("setEnv(%q) failed: %v", key, err)
		}
	}
}

func TestYQSetterClone(t *testing.T) {
	base := yqSetter{}
	base.setEnv("A", "1")

	vm := base.clone()
	vm.setEnv("B", "2")

	if len(base.exprs) != 1 {
		t.Errorf("clone changed the original setter: %+v", base)
	}
	if len(vm.exprs) != 2 {
		t.Errorf("clone = %+v, want 2 entries", vm)
	}
}

func TestYQSetterSetEnvDefault(t *testing.T) {
	s := yqSetter{}
	if err := s.setEnvDefault("SHIKARI_SCENARIO_NAME", "nomad"); err != nil {
		t.Fatal(err)
	}

	want := `.env.SHIKARI_SCENARIO_NAME = (.env.SHIKARI_SCENARIO_NAME // ("bm9tYWQ=" | @base64d))`
	if expr := s.expression(); expr != want {
		t.Errorf("expression = %q, want %q", expr, want)
	}

	if err := s.setEnvDefault("BAD KEY", "x"); err == nil {
		t.Error("setEnvDefault accepted an invalid key")
	}
}
