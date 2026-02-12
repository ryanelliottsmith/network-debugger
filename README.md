# Network Debugger

> [!IMPORTANT]
This is a WIP. A lot of functionality is either entirely missing, or incomplete

A comprehensive network debugging tool for Kubernetes clusters (RKE2/K3s). Helps diagnose connectivity issues, DNS problems, port accessibility, bandwidth constraints, and host configuration issues.

## Features

- **Coordinated Testing**: Deploys two DaemonSets (one with hostNetwork, one on overlay network) and runs tests across all cluster nodes
- **Dual Network Path Testing**: Connectivity checks (ping, ports, bandwidth) run on both hostNetwork and overlay (CNI) pods to test both network paths
- **Comprehensive Checks**:
  - DNS resolution (cluster DNS, external DNS, custom servers)
  - ICMP ping connectivity with latency stats
  - TCP/UDP port connectivity (default RKE2/K3s ports + custom)
  - Bandwidth testing (iperf3-based)
  - Host configuration (IP forwarding, MTU, kernel params)
  - Conntrack statistics and failure detection
  - iptables/nftables duplicate rule detection
- **Multiple Output Formats**: Table, JSON, YAML


## Check Types

### Connectivity Checks
These checks test network connectivity between nodes:
- **ping**: ICMP connectivity with latency statistics (runs on both host and overlay networks)
- **bandwidth**: iperf3-based throughput testing (runs on both host and overlay networks)
- **ports**: TCP/UDP port accessibility (runs on **host network only**)

### Node-Wide Checks
These checks run on all pods (both networks) across all nodes:
- **dns**: DNS resolution testing
  - **Host Network**: Tests external resolution (e.g. google.com)
  - **Overlay Network**: Tests internal (cluster.local) and external resolution

### Local Configuration Checks
These checks examine local node configuration and **run once per node** (hostNetwork pod only):
- **hostconfig**: IP forwarding, MTU, kernel parameters
- **conntrack**: Connection tracking statistics and capacity
- **iptables** (WIP): Detects duplicate iptables/nftables rules and backend conflicts

## Installation

### Build from Source

```bash
make build
sudo make install
```

## Quick Start

### Coordinated Cluster Testing

The `netdebug run` command handles the full lifecycle of the test:
1.  **Deploys** the DaemonSets if they are not already present.
2.  **Runs** the specified checks.
3.  **Cleans up** the DaemonSets automatically after the run (unless `--cleanup=false` is used).

```bash
# Run all default checks (excludes bandwidth)
netdebug run

# Run specific checks
netdebug run --checks=dns,ping,ports

# Include bandwidth testing
netdebug run --checks=dns,ping,bandwidth

# Test only overlay network
netdebug run --no-host-network

# Keep DaemonSets running after completion (useful for debugging)
netdebug run --cleanup=false
```

### DaemonSet Management

```bash
# Deploy DaemonSet
netdebug deploy install

# Check status
netdebug deploy status

# Remove DaemonSet
netdebug deploy uninstall
```

## Architecture

### Coordinated Mode

```
CLI (netdebug run)
  ├─ Deploys two DaemonSets (host + overlay network)
  ├─ Waits for all pods to be ready
  ├─ Updates ConfigMap with test configuration
  ├─ Watches pod logs for structured JSON events
  ├─ Aggregates results from all nodes
  └─ Displays final report

DaemonSet Pods (netdebug agent --mode=configmap)
  ├─ Watch ConfigMap for test triggers
  ├─ Execute configured checks
  ├─ Emit structured JSON log events
  └─ Return to watching for next run
```

## Default Ports

The tool includes default port checks for RKE2/K3s clusters. These checks are context-aware and verify ports based on the target node's role (Control Plane vs. Worker).

### Currently Tested Ports

| Port | Protocol | Service | Role | Status |
|------|----------|---------|------|--------|
| 10250 | TCP | kubelet | All Nodes | ✅ Enabled |
| 6443 | TCP | kube-apiserver | Control Plane | ✅ Enabled |
| 9345 | TCP | RKE2 supervisor | Control Plane | ✅ Enabled |
| 2379 | TCP | etcd-client | Control Plane | ✅ Enabled |
| 2380 | TCP | etcd-peer | Control Plane | ✅ Enabled |

### Available But Disabled Ports

The following ports are defined but currently disabled in the default configuration:

**Kubernetes Core:**
| Port | Protocol | Service | Notes |
|------|----------|---------|-------|
| 10255 | TCP | kubelet-readonly | Deprecated in newer versions |

**CNI - Flannel:**
| Port | Protocol | Service |
|------|----------|---------|
| 8472 | UDP | VXLAN |
| 51820 | UDP | WireGuard IPv4 |
| 51821 | UDP | WireGuard IPv6 |

**CNI - Calico:**
| Port | Protocol | Service |
|------|----------|---------|
| 179 | TCP | BGP |

**CNI - Cilium:**
| Port | Protocol | Service |
|------|----------|---------|
| 4240 | TCP | Cilium health |

To enable additional ports, see `pkg/types/ports.go` and uncomment the desired ports in the `DefaultPorts()` function.

Override defaults with: `--ports=8080/tcp:myapp,9000/udp:custom`

## Output Formats

### Table (default)

```
┌────────┬─────────┬──────────┬────────┬──────────────────┐
│ Source │ Target  │ Check    │ Status │ Details          │
├────────┼─────────┼──────────┼────────┼──────────────────┤
│ node-1 │ node-2  │ ping     │ ✓      │ 0.45ms avg       │
│ node-1 │ node-3  │ ping     │ ✗      │ timeout after 5s │
└────────┴─────────┴──────────┴────────┴──────────────────┘
```

### JSON

```bash
netdebug run --output=json > results.json
```

### YAML

```bash
netdebug run --output=yaml > results.yaml
```

## Development

### Prerequisites

- Go 1.23+
- Docker (for container builds)
- golangci-lint (for linting)

### Building

```bash
# Build binary
make build

# Run tests
make test

# Lint code
make lint

# Build Docker image
make docker-build
```

### Project Structure

```
network-debugger/
├── cmd/netdebug/          # CLI entry point
│   └── commands/          # Cobra commands
├── pkg/
│   ├── checks/            # Test implementations
│   ├── agent/             # Agent mode logic
│   ├── coordinator/       # CLI coordination
│   ├── k8s/               # Kubernetes integration
│   ├── types/             # Shared types
│   └── output/            # Output formatting
├── internal/manifests/    # Embedded K8s manifests
└── Dockerfile             # Multi-arch container
```

## Contributing

Contributions welcome! Please open an issue or PR.

## License

[License TBD]
