package shikari

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// yqSetter builds the yq expression passed to `limactl start --set`.
//
// Values are never written into the expression as they are. Each one is
// base64 encoded and decoded by yq with @base64d, so values containing quotes,
// backslashes or newlines (eg: license files) need no escaping and cannot
// change the expression.
//
// strenv() can't be used: Lima v2.1.0 and later disable yq's env operations
// for --set, and fail with "env operations have been disabled".
type yqSetter struct {
	exprs []string
}

// literal returns a yq expression that evaluates to value as a string.
func literal(value string) string {
	return fmt.Sprintf(`("%s" | @base64d)`, base64.StdEncoding.EncodeToString([]byte(value)))
}

// set assigns value to the yq path, eg: .env.SHIKARI_CLUSTER_NAME
func (s *yqSetter) set(path string, value string) {
	s.exprs = append(s.exprs, fmt.Sprintf("%s = %s", path, literal(value)))
}

// setEnv sets an environment variable in the Lima config of the VM.
func (s *yqSetter) setEnv(key string, value string) error {
	if !envKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid environment variable name %q", key)
	}

	s.set(fmt.Sprintf(".env.%s", key), value)
	return nil
}

// setEnvDefault sets an environment variable in the Lima config of the VM,
// unless the template already sets it.
func (s *yqSetter) setEnvDefault(key string, value string) error {
	if !envKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid environment variable name %q", key)
	}

	path := fmt.Sprintf(".env.%s", key)
	s.exprs = append(s.exprs, fmt.Sprintf("%s = (%s // %s)", path, path, literal(value)))
	return nil
}

func (s yqSetter) expression() string {
	return strings.Join(s.exprs, " | ")
}

// clone returns a copy that can be extended without changing s.
func (s yqSetter) clone() yqSetter {
	return yqSetter{
		exprs: append([]string{}, s.exprs...),
	}
}
