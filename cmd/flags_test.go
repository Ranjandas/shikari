package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNameFlag(t *testing.T) {
	for _, tt := range []struct {
		cmd      *cobra.Command
		required bool
	}{
		{createCmd, true}, {destroyCmd, true}, {envCmd, true}, {execCmd, true},
		{scaleCmd, true}, {startCmd, true}, {stopCmd, true}, {listCmd, false},
	} {
		flag := tt.cmd.Flags().Lookup("name")
		if flag == nil || flag.Shorthand != "n" {
			t.Errorf("%s: no --name/-n flag", tt.cmd.Name())
			continue
		}

		_, required := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if required != tt.required {
			t.Errorf("%s: --name required = %v, want %v", tt.cmd.Name(), required, tt.required)
		}
	}
}
