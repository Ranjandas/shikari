package cmd

import (
	"errors"
	"reflect"
	"testing"

	lima "github.com/ranjandas/shikari/app/lima"
)

func names(vms []lima.LimaVM) []string {
	var n []string
	for _, v := range vms {
		n = append(n, v.Name)
	}
	return n
}

func TestExecTargets(t *testing.T) {
	var instances []lima.LimaVM
	for _, n := range []string{"m-srv-01", "m-srv-11", "m-cli-01", "m-cli-02"} {
		instances = append(instances, lima.LimaVM{Name: n})
	}

	tests := []struct {
		name             string
		servers, clients bool
		instance         string
		want             []string
		wantErr          bool
	}{
		{"all", false, false, "", []string{"m-srv-01", "m-srv-11", "m-cli-01", "m-cli-02"}, false},
		{"servers", true, false, "", []string{"m-srv-01", "m-srv-11"}, false},
		{"clients", false, true, "", []string{"m-cli-01", "m-cli-02"}, false},
		{"instance", false, false, "srv-01", []string{"m-srv-01"}, false},
		{"instance without padding", false, false, "srv-1", []string{"m-srv-01"}, false},
		{"instance with cluster name", false, false, "m-cli-02", []string{"m-cli-02"}, false},
		{"partial suffix is not a match", false, false, "1", nil, true},
		{"unknown instance", false, false, "cli-05", nil, true},
	}

	for _, tt := range tests {
		got, err := execTargets(instances, "m", tt.servers, tt.clients, tt.instance)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if g := names(got); !reflect.DeepEqual(g, tt.want) {
			t.Errorf("%s: execTargets = %v, want %v", tt.name, g, tt.want)
		}
	}
}

func TestExitCode(t *testing.T) {
	if got := exitCode(errors.New("plain")); got != 1 {
		t.Errorf("exitCode(plain) = %d, want 1", got)
	}

	if got := exitCode(&exitCodeError{code: 3, err: errors.New("remote")}); got != 3 {
		t.Errorf("exitCode(exitCodeError) = %d, want 3", got)
	}

	if got := exitCode(&exitCodeError{code: 0, err: errors.New("not an exit error")}); got != 1 {
		t.Errorf("exitCode(code 0) = %d, want 1", got)
	}
}
