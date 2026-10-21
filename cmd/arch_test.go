package cmd

import (
	"testing"
)

func TestLimaArch(t *testing.T) {
	tests := map[string]string{"arm64": "aarch64", "amd64": "x86_64", "s390x": "s390x", "riscv64": "riscv64"}

	for goarch, want := range tests {
		if got := limaArch(goarch); got != want {
			t.Errorf("limaArch(%q) = %q, want %q", goarch, got, want)
		}
	}
}
