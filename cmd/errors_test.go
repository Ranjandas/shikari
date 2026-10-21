package cmd

import (
	"errors"
	"testing"
)

func TestCollectErrors(t *testing.T) {
	errCh := make(chan error, 2)
	if err := collectErrors(errCh); err != nil {
		t.Errorf("collectErrors with no errors = %v", err)
	}

	first, second := errors.New("first"), errors.New("second")
	errCh = make(chan error, 2)
	errCh <- first
	errCh <- second

	err := collectErrors(errCh)
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Errorf("collectErrors = %v, want both errors", err)
	}
}
