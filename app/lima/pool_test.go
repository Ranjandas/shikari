package lima

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunAllLimitsParallelism(t *testing.T) {
	var running, peak atomic.Int32

	err := RunAll([]string{"a", "b", "c", "d", "e", "f"}, 2, 0, func(string) error {
		n := running.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		running.Add(-1)
		return nil
	})

	if err != nil {
		t.Fatalf("RunAll = %v", err)
	}
	if got := peak.Load(); got != 2 {
		t.Errorf("peak parallelism = %d, want 2", got)
	}
}

func TestRunAllRetriesAndJoinsErrors(t *testing.T) {
	retryDelay = 0
	t.Cleanup(func() { retryDelay = 3 * time.Second })

	var mu sync.Mutex
	calls := map[string]int{}
	errBroken := errors.New("broken")

	err := RunAll([]string{"flaky", "broken", "ok"}, 3, 1, func(name string) error {
		mu.Lock()
		calls[name]++
		n := calls[name]
		mu.Unlock()

		switch {
		case name == "flaky" && n == 1:
			return errors.New("transient")
		case name == "broken":
			return errBroken
		}
		return nil
	})

	if !errors.Is(err, errBroken) {
		t.Errorf("RunAll = %v, want the broken VM's error", err)
	}
	if err != nil && err.Error() != "broken" {
		t.Errorf("RunAll = %q, want only the error that survived retries", err)
	}

	want := map[string]int{"flaky": 2, "broken": 2, "ok": 1}
	for name, n := range want {
		if calls[name] != n {
			t.Errorf("%s called %d times, want %d", name, calls[name], n)
		}
	}
}

func TestParallelism(t *testing.T) {
	for value, want := range map[string]int{"": DefaultParallelism, "1": 1, "5": 5, "0": DefaultParallelism, "x": DefaultParallelism} {
		t.Setenv("SHIKARI_PARALLELISM", value)
		if got := Parallelism(); got != want {
			t.Errorf("SHIKARI_PARALLELISM=%q: Parallelism() = %d, want %d", value, got, want)
		}
	}
}
