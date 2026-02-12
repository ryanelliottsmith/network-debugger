package types

import "time"

type TestResult struct {
	Node      string                 `json:"node"`
	Network   string                 `json:"network"`
	Check     string                 `json:"check"`
	Target    string                 `json:"target,omitempty"`
	Status    ResultStatus           `json:"status"`
	Error     string                 `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
}

type ResultStatus string

const (
	StatusPass       ResultStatus = "pass"
	StatusFail       ResultStatus = "fail"
	StatusIncomplete ResultStatus = "incomplete"
	StatusSkipped    ResultStatus = "skipped"
)

type TestSummary struct {
	TotalTests int           `json:"total_tests"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Incomplete int           `json:"incomplete"`
	Skipped    int           `json:"skipped"`
	Results    []TestResult  `json:"results"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	Duration   time.Duration `json:"duration"`
}

type DNSCheckDetails struct {
	Server      string   `json:"server"`
	Query       string   `json:"query"`
	ResolvedIPs []string `json:"resolved_ips,omitempty"`
	LatencyMS   float64  `json:"latency_ms"`
}

type PingCheckDetails struct {
	PacketsSent     int     `json:"packets_sent"`
	PacketsReceived int     `json:"packets_received"`
	PacketLoss      float64 `json:"packet_loss_percent"`
	MinLatencyMS    float64 `json:"min_latency_ms"`
	AvgLatencyMS    float64 `json:"avg_latency_ms"`
	MaxLatencyMS    float64 `json:"max_latency_ms"`
	TTL             int     `json:"ttl"`
}

type PortCheckDetails struct {
	Port         int     `json:"port"`
	Protocol     string  `json:"protocol"`
	Open         bool    `json:"open"`
	LatencyMS    float64 `json:"latency_ms,omitempty"`
	ResponseData string  `json:"response_data,omitempty"`
	Error        string  `json:"error,omitempty"`
}

type BandwidthCheckDetails struct {
	BandwidthMbps float64 `json:"bandwidth_mbps"`
	Retransmits   int     `json:"retransmits"`
	Protocol      string  `json:"protocol"`
	Duration      int     `json:"duration_seconds"`
}

type HostConfigDetails struct {
	IPForwarding bool              `json:"ip_forwarding"`
	MTU          int               `json:"mtu"`
	KernelParams map[string]string `json:"kernel_params,omitempty"`
	Issues       []string          `json:"issues,omitempty"`
}

type ConntrackDetails struct {
	Entries       int      `json:"entries"`
	MaxEntries    int      `json:"max_entries"`
	InsertsFailed int      `json:"inserts_failed"`
	DropCount     int      `json:"drop_count"`
	Issues        []string `json:"issues,omitempty"`
}

type IptablesDetails struct {
	LegacyRuleCount  int      `json:"legacy_rule_count"`
	NftableRuleCount int      `json:"nftable_rule_count"`
	DuplicateRules   int      `json:"duplicate_rules"`
	Issues           []string `json:"issues,omitempty"`
}
