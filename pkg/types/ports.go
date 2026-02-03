package types

type NodeRole string

const (
	// NodeRoleAll indicates the port should be checked on all nodes
	NodeRoleAll NodeRole = "all"
	// NodeRoleControlPlane indicates the port should only be checked on control plane nodes
	NodeRoleControlPlane NodeRole = "controlplane"
)

type PortCheck struct {
	Port     int      `json:"port"`
	Protocol string   `json:"protocol"`
	Name     string   `json:"name"`
	NodeRole NodeRole `json:"node_role"`
}

func DefaultPorts() []PortCheck {
	return []PortCheck{

		{Port: 10250, Protocol: "tcp", Name: "kubelet", NodeRole: NodeRoleAll},

		{Port: 6443, Protocol: "tcp", Name: "kube-apiserver", NodeRole: NodeRoleControlPlane},
		{Port: 9345, Protocol: "tcp", Name: "rke2-supervisor", NodeRole: NodeRoleControlPlane},

		{Port: 2379, Protocol: "tcp", Name: "etcd-client", NodeRole: NodeRoleControlPlane},
		{Port: 2380, Protocol: "tcp", Name: "etcd-peer", NodeRole: NodeRoleControlPlane},
	}
}

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
