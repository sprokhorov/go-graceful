# go-graceful

[![Go Reference](https://pkg.go.dev/badge/github.com/sprokhorov/go-graceful.svg)](https://pkg.go.dev/github.com/sprokhorov/go-graceful)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A lightweight and ergonomic Go package to coordinate the graceful shutdown of concurrent processes, background workers, and HTTP servers.

## Features

- **Context-driven:** Integrates seamlessly with standard library `context.Context` for cancellation.
- **Wait coordination:** Uses `sync.WaitGroup` internally to block until all registered goroutines have exited.
- **OS Signal handling:** Out-of-the-box support for capturing termination signals like `SIGINT` (Ctrl+C) and `SIGTERM`.
- **Timeout control:** Enforces a hard limit on cleanup time to prevent processes from hanging indefinitely.

## Installation

```bash
go get github.com/sprokhorov/go-graceful
```

## Quick Start

Here is a simple example showing how to coordinate graceful shutdown for a HTTP server and a background worker:

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sprokhorov/go-graceful"
)

func main() {
	// 1. Initialize ShutdownManager
	mgr := graceful.New()

	// 2. Start listening for SIGINT (Ctrl+C) and SIGTERM in the background
	mgr.ListenForSignals()

	// 3. Register a background worker
	mgr.Go(func(ctx context.Context) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Background worker stopping...")
				// Perform cleanup here...
				time.Sleep(500 * time.Millisecond)
				fmt.Println("Background worker stopped.")
				return
			case <-ticker.C:
				fmt.Println("Working...")
			}
		}
	})

	// 4. Register an HTTP server
	server := &http.Server{Addr: ":8080"}
	
	mgr.Go(func(ctx context.Context) {
		// Start server in another goroutine so we can block on shutdown here
		go func() {
			fmt.Println("Starting HTTP server on :8080")
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				fmt.Printf("HTTP server ListenAndServe error: %v\n", err)
			}
		}()

		// Block until context is canceled (SIGINT/SIGTERM or manual cancel)
		<-ctx.Done()
		fmt.Println("Shutting down HTTP server...")

		// Enforce a timeout for server shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("HTTP server forced shutdown: %v\n", err)
		} else {
			fmt.Println("HTTP server stopped gracefully.")
		}
	})

	// 5. Wait for all registered goroutines to finish, with a max timeout of 10s
	fmt.Println("Application running. Press Ctrl+C to stop.")
	mgr.Wait(10 * time.Second)
	fmt.Println("Application exited.")
}
```

## API Reference

### `New() *ShutdownManager`
Initializes a new `ShutdownManager` with a background context and cancellation handler.

### `Context() context.Context`
Returns the shared context. Goroutines should poll this context's `Done()` channel to detect shutdown.

### `Go(f func(ctx context.Context))`
Registers and starts a new goroutine managed by the `ShutdownManager`. The manager increments its internal `WaitGroup` before starting the function, and decrements it when the function returns.

### `ListenForSignals()`
Starts a background goroutine that listens for `SIGINT` (Ctrl+C) and `SIGTERM`. When a signal is received, the shared context is canceled.

### `Wait(timeout time.Duration)`
Blocks the execution until all goroutines launched via `Go` have finished.
- If `timeout > 0`, it will wait at most for the specified duration. If the duration expires before all goroutines finish, it returns anyway.
- If `timeout <= 0`, it waits indefinitely.

## License

Released under the [Apache 2.0 License](LICENSE).
