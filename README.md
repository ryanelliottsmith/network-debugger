# Network Debugger

A comprehensive network debugging tool for Kubernetes clusters (RKE2/K3s). Helps diagnose connectivity issues, DNS problems, port accessibility, bandwidth constraints, and host configuration issues.

## Features

- **Coordinated Testing**: Deploy as DaemonSet and run tests across all cluster nodes
- **Multiple Network Paths**: Test both hostNetwork and overlay (CNI) network paths
- **Comprehensive Checks**:
  - DNS resolution (cluster DNS, external DNS, custom servers)
  - ICMP ping connectivity with latency stats
  - TCP/UDP port connectivity (default RKE2/K3s ports + custom)
  - Bandwidth testing (iperf3-based)
  - Host configuration (IP forwarding, MTU, kernel params)
  - Conntrack statistics and failure detection
  - iptables/nftables duplicate rule detection
- **Flexible Deployment**: Run as coordinated DaemonSet or standalone container
- **Multiple Output Formats**: Table, JSON, YAML

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

### Standalone Checks

Run individual checks locally:

```bash
# DNS check
netdebug check dns --servers=8.8.8.8,1.1.1.1

# Ping test
netdebug check ping --targets=10.0.0.1,10.0.0.2

# Port connectivity
netdebug check ports --targets=10.0.0.1 --ports=6443/tcp:api,10250/tcp:kubelet

# Host configuration
netdebug check hostconfig

# Conntrack statistics
netdebug check conntrack

# iptables check
netdebug check iptables

# Bandwidth test
netdebug check bandwidth --target=10.0.0.1 --duration=10s
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

### Standalone Mode

Run the container directly for local testing:

```bash
# Using kubectl
kubectl run netdebug --rm -it \
  --image=ghcr.io/ryanelliottsmith/network-debugger:latest \
  -- agent --checks=dns,hostconfig

# Using Docker
docker run --rm \
  ghcr.io/ryanelliottsmith/network-debugger:latest \
  agent --checks=hostconfig,conntrack
```

## Default Ports

The tool includes default port checks for RKE2/K3s clusters:

| Port | Protocol | Service |
|------|----------|---------|
| 6443 | TCP | kube-apiserver |
| 10250 | TCP | kubelet |
| 9345 | TCP | RKE2 supervisor |
| 2379-2380 | TCP | etcd |
| 8472 | UDP | VXLAN (Flannel) |
| 51820-51821 | UDP | WireGuard |
| 179 | TCP | BGP (Calico) |
| 4240 | TCP | Cilium health |

Override with `--ports=8080/tcp:myapp,9000/udp:custom`

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
