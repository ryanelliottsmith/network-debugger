# Network Debugger

A CLI tool for diagnosing network connectivity and configuration issues in Kubernetes clusters (RKE2/K3s). It automates the deployment of test DaemonSets, executes checks across nodes, and aggregates the results.

## Prerequisites

- A valid Kubeconfig (`~/.kube/config` or set via `KUBECONFIG` environment variable)
- Cluster RBAC permissions to create `Namespace`, `ServiceAccount`, `ClusterRole`, `ClusterRoleBinding`, `DaemonSet`, and `ConfigMap`.

## Installation

Download the latest pre-compiled binary from GitHub releases:

```bash
# Example for Linux AMD64
curl -sL https://github.com/ryanelliottsmith/network-debugger/releases/latest/download/netdebug-linux-amd64 -o netdebug
chmod +x netdebug
```

## Usage

The CLI operates by dynamically deploying a host-network and an overlay-network DaemonSet, passing configuration via a ConfigMap, and reading JSON events from pod logs to aggregate results.

### Coordinated Test Execution

The `run` subcommand handles deployment, test execution, and cleanup automatically.

Execute the default check suite (dns, ping, ports, conntrack, hostconfig):

```bash
./netdebug run
```

Execute specific checks:

```bash
./netdebug run --checks=dns,ping,bandwidth
```

Execute a bandwidth test with custom `iperf3` arguments, disable the overlay network tests, and retain the DaemonSets for future runs:

```bash
./netdebug run --checks=bandwidth --overlay=false --cleanup=false --iperf-args="-t 120"
```

Output formats can be modified using the `-o` or `--output` flag (`table`, `json`, `yaml`):

```bash
./netdebug run --checks=dns -o json
```

Deploy custom image:

```bash
./bin/netdebug run --image ghcr.io/ryanelliottsmith/network-debugger:dev-e7516c4
```
The `--image` tag is also accepted by `netdebug deploy install`

### Manual DaemonSet Lifecycle Management

For granular control, the DaemonSet lifecycle can be managed manually.

Install the DaemonSets:

```bash
./netdebug deploy install
```

Check deployment status:

```bash
./netdebug deploy status
```

Remove the DaemonSets and associated RBAC resources:

```bash
./netdebug deploy uninstall
```

### Advanced: Customizing Manifests (Template)

To modify the default Kubernetes manifests (e.g., adding custom `tolerations`, `nodeSelector`, or labels), use the `template` command to output the base YAML:

```bash
./netdebug deploy template > manifests.yaml
```

Modify `manifests.yaml` as needed, then apply it manually:

```bash
kubectl apply -f manifests.yaml
```

> **Note:** The `netdebug` CLI relies on specific names and labels to orchestrate tests. When modifying the templates, **do not change** the following:
> - DaemonSet Names: `netdebug-host` and `netdebug-overlay`
> - ConfigMap Name: `netdebug-config`
> - Pod Labels: `app: netdebug` and `network-mode: [host|overlay]`

### Check Definitions

- `ping`: ICMP latency and loss between nodes (Host and overlay networks).
- `bandwidth`: TCP throughput via `iperf3` (Host and overlay networks).
- `ports`: TCP accessibility for control plane and worker node default ports (Host only).
- `dns`: Resolution for `cluster.local` and external addresses (Host and overlay networks).
- `hostconfig`: IP forwarding, MTU verification, and sysctl parameters.
- `conntrack`: Connection tracking table utilization.
- `iptables`: (WIP) Detects duplicate rules.
