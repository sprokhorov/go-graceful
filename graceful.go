// Package graceful provides tools for coordinating the graceful shutdown of concurrent processes
// and goroutines, supporting timeout controls and OS signal listening.
package graceful

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownManager coordinates the graceful shutdown of goroutines.
// It manages a shared context that is canceled when a shutdown is initiated,
// and a wait group to track active goroutines.
type ShutdownManager struct {
	wg     sync.WaitGroup     // Coordinates the termination of registered goroutines
	ctx    context.Context    // Shared context passed to goroutines, canceled on shutdown
	cancel context.CancelFunc // Cancels the shared context
}

// New creates a shutdown manager with background context
func New() *ShutdownManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ShutdownManager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Context returns the shared cancellation context
func (s *ShutdownManager) Context() context.Context {
	return s.ctx
}

// Go starts a goroutine that participates in shutdown coordination
func (s *ShutdownManager) Go(f func(ctx context.Context)) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		f(s.ctx)
	}()
}

// Wait waits for all goroutines to exit, with optional timeout.
// If timeout <= 0, waits forever.
func (s *ShutdownManager) Wait(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	if timeout > 0 {
		select {
		case <-done:
			// graceful shutdown completed
		case <-time.After(timeout):
			// shutdown timeout reached
		}
	} else {
		<-done
	}
}

// ListenForSignals listens to SIGINT/SIGTERM and cancels context on signal
func (s *ShutdownManager) ListenForSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		s.cancel()
	}()
}
