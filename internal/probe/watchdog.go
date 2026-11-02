package probe

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// WatchdogPool coordinates bounded concurrent tasks with hard timeouts and panic isolation.
type WatchdogPool struct {
	concurrency int
	timeout     time.Duration
	sem         chan struct{}
}

// NewWatchdogPool creates a worker pool bounded to maxConcurrency and taskTimeout.
func NewWatchdogPool(maxConcurrency int, taskTimeout time.Duration) *WatchdogPool {
	if maxConcurrency <= 0 {
		maxConcurrency = 16
	}
	if taskTimeout <= 0 {
		taskTimeout = 3 * time.Second
	}
	return &WatchdogPool{
		concurrency: maxConcurrency,
		timeout:     taskTimeout,
		sem:         make(chan struct{}, maxConcurrency),
	}
}

// Execute runs fn with a worker slot and timeout. Panics inside fn are caught safely.
func (p *WatchdogPool) Execute(ctx context.Context, fn func(taskCtx context.Context) error) error {
	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
	case <-ctx.Done():
		return ctx.Err()
	}

	taskCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("task panicked: %v\nstack: %s", r, debug.Stack())
			}
		}()
		errChan <- fn(taskCtx)
	}()

	select {
	case err := <-errChan:
		return err
	case <-taskCtx.Done():
		return fmt.Errorf("task exceeded hard watchdog timeout (%v): %w", p.timeout, taskCtx.Err())
	}
}

// BatchExecute runs multiple tasks across the bounded worker pool.
func (p *WatchdogPool) BatchExecute(ctx context.Context, tasks []func(ctx context.Context) error) []error {
	results := make([]error, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		idx := i
		t := task
		go func() {
			defer wg.Done()
			results[idx] = p.Execute(ctx, t)
		}()
	}

	wg.Wait()
	return results
}
