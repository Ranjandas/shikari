package cmd

import "runtime"

// defaultArch returns the Lima architecture name for the host the CLI was
// built for. Lima names the two common hosts aarch64 and x86_64; any other
// host keeps the previous aarch64 default.
func defaultArch() string {
	return archFor(runtime.GOARCH)
}

// archFor maps a Go architecture to its Lima name.
func archFor(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return "aarch64"
	}
}
