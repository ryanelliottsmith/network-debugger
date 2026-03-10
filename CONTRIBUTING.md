# Contributing to Network Debugger

## Getting Started

To set up your local development environment:

```bash
git clone https://github.com/ryanelliottsmith/network-debugger.git
cd network-debugger
go mod tidy
make build
```

## Running Locally

You can run the CLI tool locally using the built binary or directly with `go run`:

```bash
make build && ./bin/netdebug
# or
go run ./cmd/netdebug
```

**Note on Cluster Testing:** Testing the DaemonSet component requires building a container image and making it available to your cluster nodes. You can do this by building and pushing the image.

You can override the `IMAGE_NAME` and `IMAGE_TAG` variables to point to your own container registry:

```bash
make docker-build IMAGE_NAME=myregistry/network-debugger IMAGE_TAG=test
make docker-push IMAGE_NAME=myregistry/network-debugger IMAGE_TAG=test
```

Alternatively, you can sideload the built image directly to your cluster nodes for testing.

## Formatting & Linting

Before committing your changes, you must format and lint your code. These steps are mandatory:

```bash
make fmt
make lint
```


## Adding a New Check

Network Debugger is designed to be extensible. All checks implement the `type Check interface` defined in `pkg/checks/check.go`.

### The `Check` Interface

The interface consists of 7 methods:

1. `Name() string`: Returns the unique name of the check.
2. `Description() string`: Returns a short description of the check to be displayed in the CLI output.
3. `Run(ctx context.Context, target string) (*types.TestResult, error)`: Executes the check logic against a target.
4. `IsLocal() bool`: Returns `true` if the check runs locally and doesn't have a meaningful target (hides the Target column in table output).
5. `HostNetworkOnly() bool`: Returns `true` if the check requires the host network namespace (e.g., inspecting iptables, conntrack).
6. `AlwaysShow() bool`: Returns `true` if the check should always be displayed in the output, even when passing.
7. `FormatSummary(details interface{}, debug bool) string`: Formats the check details for display in the table output's Details column.

### Example: Hello Check

Here is a minimal example of a new check implementation:

```go
package checks

import (
 "context"
 "fmt"

 "github.com/ryanelliottsmith/network-debugger/pkg/types"
)

type HelloCheck struct{}

func (c *HelloCheck) Name() string {
    return "hello"
}

func (c *HelloCheck) Description() string {
    return "Tests basic hello connectivity to a target."
}

func (c *HelloCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
    result := &types.TestResult{
        Check:  c.Name(),
        Target: target,
        Status: types.StatusPass,
    }

    // Simple check logic
    message := fmt.Sprintf("Hello from %s", target)
    
    if result.Details == nil {
        result.Details = make(map[string]interface{})
    }
    result.Details["hello"] = message

    return result, nil
}

func (c *HelloCheck) IsLocal() bool {
    return true
}

func (c *HelloCheck) HostNetworkOnly() bool {
    return false
}

func (c *HelloCheck) AlwaysShow() bool {
    return true
}

func (c *HelloCheck) FormatSummary(details interface{}, debug bool) string {
    if details == nil {
        return ""
    }

    switch d := details.(type) {
    case map[string]interface{}:
        if msg, ok := d["hello"].(string); ok {
            return msg
        }
    }

    return ""
}

func NewHelloCheck() *HelloCheck {
    return &HelloCheck{}
}

// Register the check automatically
func init() {
    DefaultRegistry.Register(NewHelloCheck())
}
```

### Check Result Formatting

When your check finishes its `Run` method, it returns a `*types.TestResult` which includes a `Details` field (typically a `map[string]interface{}`). This field holds the raw result data of your network check.

To display this data concisely in the CLI's table output, you must implement the `FormatSummary(details interface{}, debug bool) string` method. The tool's output formatter will pass the raw `Details` object directly into this function.

**How to implement it:**
1. Type-assert the `details` interface (usually to `map[string]interface{}`).
2. Extract the relevant fields.
3. Return a human-readable string.

For example, if your `Run` method assigns `result.Details["latency"] = 45.2`, your `FormatSummary` should extract and format it:

```go
func (c *MyCheck) FormatSummary(details interface{}, debug bool) string {
    if details == nil {
        return ""
    }

    if d, ok := details.(map[string]interface{}); ok {
        if latency, ok := d["latency"].(float64); ok {
            return fmt.Sprintf("%.2fms", latency)
        }
    }
    
    return ""
}
```

### Handling Pass/Fail Status

When your `Run` method completes, you must indicate whether the network check succeeded or failed using the `Status` field on `*types.TestResult`. The status constants are defined in the `types` package:

- `types.StatusPass`: The check succeeded.
- `types.StatusFail`: The check failed (e.g., connection refused, timeout).
- `types.StatusSkipped`: The check was skipped (e.g., not applicable for this node).
- `types.StatusIncomplete`: The check could not finish.

If your check fails, you should populate the `Error` string field on the result with a human-readable explanation. This string will be printed in the output when users run the CLI:

```go
func (c *MyCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
    result := &types.TestResult{
        Check:  c.Name(),
        Target: target,
        Status: types.StatusPass, // Default to pass
    }

    err := performNetworkCheck()
    if err != nil {
        result.Status = types.StatusFail
        result.Error = fmt.Sprintf("connection refused: %v", err)
        return result, nil // Return the result with a fail status, not a Go error
    }

    return result, nil
}
```

*Note: If your `Run` method returns a non-nil Go `error` (e.g., `return nil, err`), the CLI runner will automatically catch it and convert the result to a `StatusFail` with the error message. However, the convention is to handle expected check failures (like a closed port or timeout) by returning a populated `TestResult` without a Go error.*

### Wiring Standalone Checks

While the `init()` function registers the check with the `DefaultRegistry` for DaemonSet execution, standalone checks might also need to be wired into the CLI commands. If your check can be run directly from the CLI, you may need to add it to `cmd/netdebug/commands/check.go`.

## CI/CD Pipeline

Currently, the `.github/workflows/dev.yaml` workflow automatically builds and pushes a development image (`ghcr.io/ryanelliottsmith/network-debugger:dev-<sha>`) on every commit to the `main` branch. 

**Note:** This is a temporary workflow setup and will change to proper semver in the near future.
