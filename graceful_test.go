package graceful

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	mgr := New()
	if mgr == nil {
		t.Fatal("expected New() to return a non-nil ShutdownManager")
	}
	if mgr.Context() == nil {
		t.Fatal("expected Context() to return a non-nil context")
	}
	select {
	case <-mgr.Context().Done():
		t.Fatal("context should not be canceled initially")
	default:
	}
}

func TestGoAndWait(t *testing.T) {
	mgr := New()
	runCount := 0
	ch := make(chan struct{})

	mgr.Go(func(ctx context.Context) {
		runCount++
		close(ch)
	})

	// Wait with no timeout (waits forever/until done)
	mgr.Wait(0)

	<-ch
	if runCount != 1 {
		t.Errorf("expected goroutine to have run, got runCount = %d", runCount)
	}
}

func TestWaitTimeout(t *testing.T) {
	mgr := New()

	mgr.Go(func(ctx context.Context) {
		// Simulate a long running task that doesn't respect context cancellation quickly
		time.Sleep(100 * time.Millisecond)
	})

	start := time.Now()
	// Wait with a timeout of 10 milliseconds, which is shorter than the task sleep duration
	mgr.Wait(10 * time.Millisecond)
	duration := time.Since(start)

	if duration < 10*time.Millisecond {
		t.Errorf("expected Wait to block for at least the timeout duration, but took %v", duration)
	}
	if duration > 50*time.Millisecond {
		t.Errorf("expected Wait to return close to the timeout duration (10ms), but took %v", duration)
	}
}

func TestWaitNoTimeoutCompleted(t *testing.T) {
	mgr := New()

	mgr.Go(func(ctx context.Context) {
		time.Sleep(10 * time.Millisecond)
	})

	start := time.Now()
	// Wait with a timeout of 200 milliseconds, but the task completes in 10ms
	mgr.Wait(200 * time.Millisecond)
	duration := time.Since(start)

	if duration > 50*time.Millisecond {
		t.Errorf("expected Wait to return immediately after task completion, but took %v", duration)
	}
}

func TestListenForSignals(t *testing.T) {
	mgr := New()
	mgr.ListenForSignals()

	// Find this process and send SIGINT to it
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("failed to find process: %v", err)
	}

	err = p.Signal(syscall.SIGINT)
	if err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	// Wait for context to be canceled or timeout
	select {
	case <-mgr.Context().Done():
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for signal to cancel context")
	}
}
