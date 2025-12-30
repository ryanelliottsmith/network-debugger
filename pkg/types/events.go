package types

import "time"

type EventType string

const (
	EventTypeReady      EventType = "ready"
	EventTypeTestStart  EventType = "test_start"
	EventTypeTestResult EventType = "test_result"
	EventTypeComplete   EventType = "complete"
	EventTypeError      EventType = "error"
)

type Event struct {
	Type      EventType   `json:"type"`
	Node      string      `json:"node"`
	Network   string      `json:"network,omitempty"` // "host" or "overlay"
	Pod       string      `json:"pod,omitempty"`
	Check     string      `json:"check,omitempty"`
	Target    string      `json:"target,omitempty"`
	Status    string      `json:"status,omitempty"` // "pass" or "fail"
	Error     string      `json:"error,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RunID     string      `json:"run_id,omitempty"`
}

func ReadyEvent(node, network, pod, runID string) *Event {
	return &Event{
		Type:      EventTypeReady,
		Node:      node,
		Network:   network,
		Pod:       pod,
		RunID:     runID,
		Timestamp: time.Now(),
	}
}

func TestStartEvent(node, network, pod, check, target, runID string) *Event {
	return &Event{
		Type:      EventTypeTestStart,
		Node:      node,
		Network:   network,
		Pod:       pod,
		Check:     check,
		Target:    target,
		RunID:     runID,
		Timestamp: time.Now(),
	}
}

func TestResultEvent(node, network, pod, check, target, status string, details interface{}, runID string) *Event {
	return &Event{
		Type:      EventTypeTestResult,
		Node:      node,
		Network:   network,
		Pod:       pod,
		Check:     check,
		Target:    target,
		Status:    status,
		Details:   details,
		RunID:     runID,
		Timestamp: time.Now(),
	}
}

func ErrorEvent(node, network, pod, errMsg, runID string) *Event {
	return &Event{
		Type:      EventTypeError,
		Node:      node,
		Network:   network,
		Pod:       pod,
		Error:     errMsg,
		RunID:     runID,
		Timestamp: time.Now(),
	}
}

func CompleteEvent(node, network, pod string, summary interface{}, runID string) *Event {
	return &Event{
		Type:      EventTypeComplete,
		Node:      node,
		Network:   network,
		Pod:       pod,
		Details:   summary,
		RunID:     runID,
		Timestamp: time.Now(),
	}
}
