package lima

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// DefaultParallelism is how many limactl commands run at once, unless
// SHIKARI_PARALLELISM says otherwise.
const DefaultParallelism = 3

// retryDelay is how long to wait before retrying a failed command.
var retryDelay = 3 * time.Second

// Parallelism returns how many limactl commands to run at once, from
// SHIKARI_PARALLELISM, eg: 1 to run them one after another.
func Parallelism() int {
	value := os.Getenv("SHIKARI_PARALLELISM")
	if value == "" {
		return DefaultParallelism
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		fmt.Fprintf(os.Stderr, "ignoring SHIKARI_PARALLELISM=%q, it must be a number above 0\n", value)
		return DefaultParallelism
	}

	return n
}

// RunAll calls fn for every VM name, with at most parallelism calls running at
// once, and retries each failed call up to retries times. It returns the
// errors of the calls that still failed, joined.
func RunAll(names []string, parallelism int, retries int, fn func(vmName string) error) error {
	if parallelism < 1 {
		parallelism = 1
	}

	var wg sync.WaitGroup
	slots := make(chan struct{}, parallelism)
	errs := make([]error, len(names))

	for i, name := range names {
		wg.Add(1)
		slots <- struct{}{}

		go func(i int, name string) {
			defer wg.Done()
			defer func() { <-slots }()

			err := fn(name)
			for attempt := 1; err != nil && attempt <= retries; attempt++ {
				fmt.Fprintf(os.Stderr, "%v, retrying (%d/%d)\n", err, attempt, retries)
				time.Sleep(retryDelay)
				err = fn(name)
			}
			errs[i] = err
		}(i, name)
	}

	wg.Wait()

	return errors.Join(errs...)
}

// Names returns the names of the instances.
func Names(instances []LimaVM) []string {
	names := make([]string, 0, len(instances))
	for _, instance := range instances {
		names = append(names, instance.Name)
	}

	return names
}
