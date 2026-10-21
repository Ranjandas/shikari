package cmd

import (
	"reflect"
	"testing"

	lima "github.com/ranjandas/shikari/app/lima"
)

func TestNotStopped(t *testing.T) {
	instances := []lima.LimaVM{
		{Name: "dev-srv-01", Status: "Stopped"},
		{Name: "dev-srv-02", Status: "Running"},
		{Name: "dev-cli-01", Status: "Broken"},
	}

	want := []string{"dev-srv-02 (Running)", "dev-cli-01 (Broken)"}
	if got := notStopped(instances); !reflect.DeepEqual(got, want) {
		t.Errorf("notStopped = %v, want %v", got, want)
	}

	if got := notStopped(instances[:1]); len(got) != 0 {
		t.Errorf("notStopped of stopped instances = %v", got)
	}
}
