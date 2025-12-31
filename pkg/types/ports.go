package types

type PortCheck struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp or udp
	Name     string `json:"name"`
}

func DefaultPorts() []PortCheck {
	return []PortCheck{
		// Kubernetes core
		// TODO: Separate port checks between controlplane and worker nodes
		//{Port: 6443, Protocol: "tcp", Name: "kube-apiserver"},
		{Port: 10250, Protocol: "tcp", Name: "kubelet"},
		//{Port: 10255, Protocol: "tcp", Name: "kubelet-readonly"},

		// RKE2/K3s specific
		//{Port: 9345, Protocol: "tcp", Name: "rke2-supervisor"},

		// etcd
		//{Port: 2379, Protocol: "tcp", Name: "etcd-client"},
		//{Port: 2380, Protocol: "tcp", Name: "etcd-peer"},

		// CNI - Flannel
		//{Port: 8472, Protocol: "udp", Name: "vxlan-flannel"},
		//{Port: 51820, Protocol: "udp", Name: "wireguard-ipv4"},
		//{Port: 51821, Protocol: "udp", Name: "wireguard-ipv6"},

		// CNI - Calico
		//{Port: 4789, Protocol: "udp", Name: "vxlan-calico"},
		//{Port: 179, Protocol: "tcp", Name: "bgp-calico"},
		//{Port: 5473, Protocol: "tcp", Name: "calico-typha"},

		// CNI - Cilium
		//{Port: 4240, Protocol: "tcp", Name: "cilium-health"},
		//{Port: 4244, Protocol: "tcp", Name: "hubble-peer"},
		//{Port: 4245, Protocol: "tcp", Name: "hubble-relay"},

		// Metrics
		//{Port: 10249, Protocol: "tcp", Name: "kube-proxy-metrics"},
		//{Port: 10256, Protocol: "tcp", Name: "kube-proxy-health"},
		//{Port: 10257, Protocol: "tcp", Name: "kube-controller-manager"},
		//{Port: 10259, Protocol: "tcp", Name: "kube-scheduler"},
	}
}

func ParsePortString(s string) (*PortCheck, error) {
	// TODO: Implement port string parsing
	return nil, nil
}
