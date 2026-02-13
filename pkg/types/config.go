package types

import "time"

// NetworkType indicates the network context a pod is running in.
type NetworkType string

const (
	NetworkTypeHost    NetworkType = "hostnetwork"
	NetworkTypeOverlay NetworkType = "overlay"
)

type TargetNode struct {
	NodeName       string `json:"node_name"`
	PodName        string `json:"pod_name,omitempty"`
	IP             string `json:"ip"`
	IsControlPlane bool   `json:"is_controlplane"`
}

type BandwidthTest struct {
	Active     bool   `json:"active"`
	SourceNode string `json:"source_node"`
	SourcePod  string `json:"source_pod"`
	TargetNode string `json:"target_node"`
	TargetIP   string `json:"target_ip"`
}

type Config struct {
	RunID         string         `json:"run_id"`
	TriggeredAt   time.Time      `json:"triggered_at"`
	NetworkType   NetworkType    `json:"network_type"`
	Targets       []TargetNode   `json:"targets"`
	Checks        []string       `json:"checks"`
	Ports         []PortCheck    `json:"ports"`
	DNSNames      []string       `json:"dns_names"`
	BandwidthTest *BandwidthTest `json:"bandwidth_test,omitempty"`
	Timeout       int            `json:"timeout_seconds"`
	Debug         bool           `json:"debug,omitempty"`
}
