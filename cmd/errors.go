package cmd

import (
	"errors"
)

// collectErrors closes errCh once all goroutines are done and joins the
// errors they reported, so a command fails if any of its VMs failed.
func collectErrors(errCh chan error) error {
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
