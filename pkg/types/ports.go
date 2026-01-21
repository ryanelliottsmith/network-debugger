package types

// NodeRole represents which type of nodes a port should be checked on
type NodeRole string

const (
	// NodeRoleAll indicates the port should be checked on all nodes
	NodeRoleAll NodeRole = "all"
	// NodeRoleControlPlane indicates the port should only be checked on control plane nodes
	NodeRoleControlPlane NodeRole = "controlplane"
)

// PortCheck defines a port to check connectivity against
type PortCheck struct {
	Port     int      `json:"port"`
	Protocol string   `json:"protocol"` // tcp or udp
	Name     string   `json:"name"`
	NodeRole NodeRole `json:"node_role"` // which node types have this port
}

// DefaultPorts returns the default list of ports to check
func DefaultPorts() []PortCheck {
	return []PortCheck{
		// Ports available on all nodes
		{Port: 10250, Protocol: "tcp", Name: "kubelet", NodeRole: NodeRoleAll},

		// Control plane specific ports
		{Port: 6443, Protocol: "tcp", Name: "kube-apiserver", NodeRole: NodeRoleControlPlane},
		{Port: 9345, Protocol: "tcp", Name: "rke2-supervisor", NodeRole: NodeRoleControlPlane},

		// etcd (runs on control plane nodes)
		{Port: 2379, Protocol: "tcp", Name: "etcd-client", NodeRole: NodeRoleControlPlane},
		{Port: 2380, Protocol: "tcp", Name: "etcd-peer", NodeRole: NodeRoleControlPlane},
	}
}

// FilterPortsForRole returns ports that should be checked for a target with the given role
func FilterPortsForRole(ports []PortCheck, isControlPlane bool) []PortCheck {
	var filtered []PortCheck
	for _, port := range ports {
		// Include port if:
		// 1. It's for all nodes, OR
		// 2. It's for control plane and the target is a control plane node
		if port.NodeRole == NodeRoleAll ||
			(port.NodeRole == NodeRoleControlPlane && isControlPlane) {
			filtered = append(filtered, port)
		}
	}
	return filtered
}

func ParsePortString(s string) (*PortCheck, error) {
	// TODO: Implement port string parsing
	return nil, nil
}
