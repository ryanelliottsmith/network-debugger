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

## Adding a New Check

Network Debugger is designed to be extensible. All checks implement the `type Check interface` defined in `pkg/checks/check.go`.

### The `Check` Interface

The interface consists of 6 methods:

1. `Name() string`: Returns the unique name of the check.
2. `Run(ctx context.Context, target string) (*types.TestResult, error)`: Executes the check logic against a target.
3. `IsLocal() bool`: Returns `true` if the check runs locally and doesn't have a meaningful target (hides the Target column in table output).
4. `HostNetworkOnly() bool`: Returns `true` if the check requires the host network namespace (e.g., inspecting iptables, conntrack).
5. `AlwaysShow() bool`: Returns `true` if the check should always be displayed in the output, even when passing.
6. `FormatSummary(details interface{}, debug bool) string`: Formats the check details for display in the table output's Details column.

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

### Wiring Standalone Checks

While the `init()` function registers the check with the `DefaultRegistry` for DaemonSet execution, standalone checks might also need to be wired into the CLI commands. If your check can be run directly from the CLI, you may need to add it to `cmd/netdebug/commands/check.go`.

## CI/CD Pipeline

Currently, the `.github/workflows/dev.yaml` workflow automatically builds and pushes a development image (`ghcr.io/ryanelliottsmith/network-debugger:dev-<sha>`) on every commit to the `main` branch. 

**Note:** This is a temporary workflow setup and will change in the future as the project matures.
