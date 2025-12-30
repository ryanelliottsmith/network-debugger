package agent

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// StartIperf3Server starts an iperf3 server that restarts after each client connection.
// This avoids "server busy" errors by ensuring a fresh server for each test.
// The server runs until the context is cancelled.
func StartIperf3Server(ctx context.Context) error {
	if _, err := exec.LookPath("iperf3"); err != nil {
		return fmt.Errorf("iperf3 not found in PATH: %w", err)
	}

	go runIperf3Loop(ctx)

	// Give the first server instance time to start
	time.Sleep(500 * time.Millisecond)

	return nil
}

func runIperf3Loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		cmd := exec.CommandContext(ctx, "iperf3", "-s", "--one-off")
		if err := cmd.Run(); err != nil {
			// Only log if not cancelled
			if ctx.Err() == nil {
				log.Printf("iperf3 server exited: %v", err)
			}
		}

		// Brief pause before restarting to avoid tight loop on persistent errors
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}
