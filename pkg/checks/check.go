package checks

import (
	"context"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

const (
	// DefaultCheckTimeout is the default timeout for most checks
	DefaultCheckTimeout = 5 * time.Second

	// DefaultPingTimeout is the default timeout for ping checks
	DefaultPingTimeout = 3 * time.Second

	// DefaultPortsTimeout is the default timeout for port checks
	DefaultPortsTimeout = 10 * time.Second
)

type Check interface {
	Name() string
	Run(ctx context.Context, target string) (*types.TestResult, error)
}

func RunWithTimeout(check Check, target string, timeout time.Duration) *types.TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := time.Now()
	result, err := check.Run(ctx, target)
	endTime := time.Now()

	if result == nil {
		result = &types.TestResult{
			Check:     check.Name(),
			Target:    target,
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  endTime.Sub(startTime),
		}
	}

	result.StartTime = startTime
	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = types.StatusFail
			result.Error = "timeout after " + timeout.String()
		} else {
			result.Status = types.StatusFail
			result.Error = err.Error()
		}
	}

	return result
}
