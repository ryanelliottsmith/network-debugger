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

	// IsLocal returns true for checks that run locally and don't have a meaningful target.
	// These checks won't display a Target column in table output.
	IsLocal() bool

	// AlwaysShow returns true if this check should always be displayed in output,
	// even when passing. This is useful for checks like bandwidth where the
	// result value is always interesting regardless of pass/fail status.
	AlwaysShow() bool

	// FormatSummary formats the details for display in table output.
	// Returns a human-readable summary string suitable for the Details column.
	FormatSummary(details interface{}, debug bool) string
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
