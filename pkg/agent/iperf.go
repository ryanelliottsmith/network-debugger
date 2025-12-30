package agent

import (
	"fmt"
	"os/exec"
	"time"
)

func StartIperf3Server() error {
	if _, err := exec.LookPath("iperf3"); err != nil {
		return fmt.Errorf("iperf3 not found in PATH: %w", err)
	}

	cmd := exec.Command("iperf3", "-s", "-D")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start iperf3 server: %w", err)
	}

	time.Sleep(500 * time.Millisecond)

	return nil
}
