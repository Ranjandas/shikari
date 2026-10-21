package shikari

import (
	"fmt"
	"regexp"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// yqSetter builds the yq expression passed to `limactl start --set`.
//
// Values are never written into the expression. Each one is passed to limactl
// as an environment variable and read back with strenv(), so values containing
// quotes, backslashes or newlines (eg: license files) need no escaping and
// cannot change the expression.
type yqSetter struct {
	exprs []string
	env   []string
}

// set assigns value to the yq path, eg: .env.SHIKARI_CLUSTER_NAME
func (s *yqSetter) set(path string, value string) {
	key := fmt.Sprintf("SHIKARI_YQ_VALUE_%d", len(s.env))

	s.exprs = append(s.exprs, fmt.Sprintf("%s = strenv(%s)", path, key))
	s.env = append(s.env, fmt.Sprintf("%s=%s", key, value))
}

// setEnv sets an environment variable in the Lima config of the VM.
func (s *yqSetter) setEnv(key string, value string) error {
	if !envKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid environment variable name %q", key)
	}

	s.set(fmt.Sprintf(".env.%s", key), value)
	return nil
}

func (s yqSetter) expression() string {
	return strings.Join(s.exprs, " | ")
}

// clone returns a copy that can be extended without changing s.
func (s yqSetter) clone() yqSetter {
	return yqSetter{
		exprs: append([]string{}, s.exprs...),
		env:   append([]string{}, s.env...),
	}
}
