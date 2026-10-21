package shikari

import (
	"fmt"
	"strings"
	"testing"
)

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

	want := ".env.FOO = strenv(SHIKARI_YQ_VALUE_0) | .env.FOO = strenv(SHIKARI_YQ_VALUE_1) | .env.FOO = strenv(SHIKARI_YQ_VALUE_2)"
	if expr != want {
		t.Errorf("expression = %q, want %q", expr, want)
	}

	for i, v := range values {
		if s.env[i] != fmt.Sprintf("SHIKARI_YQ_VALUE_%d=%s", i, v) {
			t.Errorf("env[%d] = %q", i, s.env[i])
		}
	}
}

func TestYQSetterRejectsInvalidEnvKeys(t *testing.T) {
	for _, key := range []string{"", "1FOO", "FOO BAR", `FOO"`, "FOO=1", "a.b", "x | .y"} {
		s := yqSetter{}
		if err := s.setEnv(key, "v"); err == nil {
			t.Errorf("setEnv(%q) did not fail", key)
		}
		if len(s.exprs) != 0 || len(s.env) != 0 {
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

	if len(base.exprs) != 1 || len(base.env) != 1 {
		t.Errorf("clone changed the original setter: %+v", base)
	}
	if len(vm.exprs) != 2 || len(vm.env) != 2 {
		t.Errorf("clone = %+v, want 2 entries", vm)
	}
}
