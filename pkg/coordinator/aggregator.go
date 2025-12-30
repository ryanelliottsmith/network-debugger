package coordinator

import (
	"sync"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

type Aggregator struct {
	mu            sync.RWMutex
	events        []*types.Event
	completedPods map[string]bool
	expectedPods  map[string]bool
	readyPods     map[string]bool
}

func NewAggregator(expectedPods []string) *Aggregator {
	expected := make(map[string]bool)
	for _, pod := range expectedPods {
		expected[pod] = true
	}

	return &Aggregator{
		events:        make([]*types.Event, 0),
		completedPods: make(map[string]bool),
		expectedPods:  expected,
		readyPods:     make(map[string]bool),
	}
}

func (a *Aggregator) AddEvent(event *types.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = append(a.events, event)

	podKey := event.Pod
	if podKey == "" {
		podKey = event.Node
	}

	switch event.Type {
	case types.EventTypeReady:
		a.readyPods[podKey] = true
	case types.EventTypeComplete:
		a.completedPods[podKey] = true
	}
}

func (a *Aggregator) AllPodsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for pod := range a.expectedPods {
		if !a.readyPods[pod] {
			return false
		}
	}
	return len(a.expectedPods) > 0
}

func (a *Aggregator) AllPodsComplete() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for pod := range a.expectedPods {
		if !a.completedPods[pod] {
			return false
		}
	}
	return len(a.expectedPods) > 0
}

func (a *Aggregator) GetEvents() []*types.Event {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*types.Event, len(a.events))
	copy(result, a.events)
	return result
}

func (a *Aggregator) GetResultEvents() []*types.Event {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var results []*types.Event
	for _, event := range a.events {
		if event.Type == types.EventTypeTestResult {
			results = append(results, event)
		}
	}
	return results
}

func (a *Aggregator) GetErrorEvents() []*types.Event {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var errors []*types.Event
	for _, event := range a.events {
		if event.Type == types.EventTypeError {
			errors = append(errors, event)
		}
	}
	return errors
}

func (a *Aggregator) GetCompletedCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.completedPods)
}

func (a *Aggregator) GetReadyCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.readyPods)
}

func (a *Aggregator) GetExpectedCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.expectedPods)
}
