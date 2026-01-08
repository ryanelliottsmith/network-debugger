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
- **Flexible Deployment**: Run as coordinated DaemonSet
- **Multiple Output Formats**: Table, JSON, YAML

## Check Types

### Connectivity Checks
These checks test network connectivity between nodes and **run on both DaemonSet pods** (hostNetwork and overlay) to test both network paths:
- **ping**: ICMP connectivity with latency statistics
- **ports**: TCP/UDP port accessibility
- **bandwidth**: iperf3-based throughput testing

### Node-Wide Checks
These checks run on all pods (both networks) across all nodes:
- **dns**: DNS resolution testing

### Local Configuration Checks
These checks examine local node configuration and **run once per node** (hostNetwork pod only):
- **hostconfig**: IP forwarding, MTU, kernel parameters
- **conntrack**: Connection tracking statistics and capacity
- **iptables**: iptables/nftables configuration and conflicts

## Installation

### Build from Source

```bash
make build
sudo make install
```

### Docker

```bash
docker pull ghcr.io/ryanelliottsmith/network-debugger:latest
```

## Quick Start

### Coordinated Cluster Testing

The tool deploys two DaemonSets to test both network paths:
- **hostNetwork DaemonSet**: Tests connectivity via the host network and runs local configuration checks
- **overlay DaemonSet**: Tests connectivity via the CNI overlay network

Connectivity checks (ping, ports, bandwidth) run on both pods to compare network performance and identify path-specific issues.

Deploy and run comprehensive tests across all nodes:

```bash
# Run all default checks (excludes bandwidth)
netdebug run

# Run specific checks
netdebug run --checks=dns,ping,ports

# Include bandwidth testing
netdebug run --checks=dns,ping,bandwidth

# Test only overlay network
netdebug run --no-host-network

# Cleanup DaemonSet after completion
netdebug run --cleanup
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

The tool includes default port checks for RKE2/K3s clusters. Most ports are currently disabled pending further testing.

### Currently Tested Ports

| Port | Protocol | Service | Status |
|------|----------|---------|--------|
| 10250 | TCP | kubelet | ✅ Enabled |

### Available But Disabled Ports

The following ports are defined but currently disabled in the default configuration:

**Kubernetes Core:**
| Port | Protocol | Service | Notes |
|------|----------|---------|-------|
| 6443 | TCP | kube-apiserver | Control plane only |
| 10255 | TCP | kubelet-readonly | Deprecated in newer versions |

**RKE2/K3s Specific:**
| Port | Protocol | Service |
|------|----------|---------|
| 9345 | TCP | RKE2 supervisor |

**etcd:**
| Port | Protocol | Service |
|------|----------|---------|
| 2379 | TCP | etcd-client |
| 2380 | TCP | etcd-peer |

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
