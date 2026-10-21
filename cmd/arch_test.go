package cmd

import "testing"

func TestArchFor(t *testing.T) {
	tests := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
		"386":   "aarch64",
		"s390x": "aarch64",
	}
	for goarch, want := range tests {
		if got := archFor(goarch); got != want {
			t.Errorf("archFor(%q) = %q, want %q", goarch, got, want)
		}
	}
}
