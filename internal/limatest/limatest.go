// Package limatest puts a fake limactl on PATH, so tests can run shikari's
// limactl calls without Lima.
package limatest

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// script records each call as its arguments separated by \x1f, prints the
// instances for `list --json`, saves the environment of `start --name <vm>`,
// and fails calls that contain a line of the fail file.
const script = `#!/bin/sh
dir=$(dirname "$0")
line=""
for arg in "$@"; do line="$line$arg$(printf '\037')"; done
printf '%s\n' "$line" >> "$dir/calls"

if [ -f "$dir/fail" ]; then
	while IFS= read -r pattern; do
		case "$line" in *"$pattern"*) echo "fake limactl: failing $*" >&2; exit 1 ;; esac
	done < "$dir/fail"
fi

case "$1" in
list) cat "$dir/instances" ;;
start) if [ "$2" = "--name" ]; then env > "$dir/env-$3"; fi ;;
esac
`

// Fake is a fake limactl installed for one test.
type Fake struct {
	t   *testing.T
	dir string
}

// Install puts a fake limactl first on PATH for the rest of the test.
// instances is what `limactl list --json` prints: one JSON object per line.
func Install(t *testing.T, instances ...string) *Fake {
	t.Helper()

	dir := t.TempDir()
	f := &Fake{t: t, dir: dir}

	f.write("limactl", script, 0o755)
	f.write("instances", strings.Join(instances, "\n")+"\n", 0o644)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return f
}

// FailOn makes every call whose arguments contain pattern exit 1, eg:
// "murphy-srv-02" or "delete".
func (f *Fake) FailOn(pattern string) {
	f.t.Helper()

	file, err := os.OpenFile(filepath.Join(f.dir, "fail"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()

	if _, err := file.WriteString(pattern + "\n"); err != nil {
		f.t.Fatal(err)
	}
}

// Calls returns the arguments of every call except `list`, sorted, as calls
// can run in parallel.
func (f *Fake) Calls() [][]string {
	f.t.Helper()

	file, err := os.Open(filepath.Join(f.dir, "calls"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(nil, 1<<20)
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "list\x1f") {
			lines = append(lines, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		f.t.Fatal(err)
	}
	sort.Strings(lines)

	calls := make([][]string, 0, len(lines))
	for _, line := range lines {
		calls = append(calls, strings.Split(strings.TrimSuffix(line, "\x1f"), "\x1f"))
	}

	return calls
}

// Env returns the environment `limactl start --name vmName` ran with.
func (f *Fake) Env(vmName string) map[string]string {
	f.t.Helper()

	data, err := os.ReadFile(filepath.Join(f.dir, "env-"+vmName))
	if err != nil {
		f.t.Fatalf("limactl start --name %s was not called: %v", vmName, err)
	}

	env := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			env[key] = value
		}
	}

	return env
}

func (f *Fake) write(name string, content string, mode os.FileMode) {
	f.t.Helper()

	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(content), mode); err != nil {
		f.t.Fatal(err)
	}
}
