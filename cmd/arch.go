package cmd

import (
	"runtime"
)

// defaultArch returns the Lima architecture matching the host, so VMs run
// natively instead of under emulation.
func defaultArch() string {
	return limaArch(runtime.GOARCH)
}

// limaArch maps a Go architecture name to the name Lima uses.
func limaArch(goarch string) string {
	switch goarch {
	case "arm64":
		return "aarch64"
	case "amd64":
		return "x86_64"
	case "arm":
		return "armv7l"
	case "riscv64", "ppc64le", "s390x":
		return goarch
	default:
		return "aarch64"
	}
}
