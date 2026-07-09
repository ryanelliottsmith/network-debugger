package types

import (
	"context"
	"sync"
)

type Check interface {
	Name() string
	Description() string
	Run(ctx context.Context, target string) (*TestResult, error)
	IsLocal() bool
	HostNetworkOnly() bool
	AlwaysShow() bool
	FormatSummary(details interface{}, quiet bool) string
}

var DefaultRegistry = NewRegistry()

type Registry struct {
	mu     sync.RWMutex
	checks map[string]Check
}

func NewRegistry() *Registry {
	return &Registry{
		checks: make(map[string]Check),
	}
}

func (r *Registry) Register(check Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[check.Name()] = check
}

func (r *Registry) Get(name string) Check {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.checks[name]
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.checks))
	for name := range r.checks {
		names = append(names, name)
	}
	return names
}

func (r *Registry) All() []Check {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checks := make([]Check, 0, len(r.checks))
	for _, check := range r.checks {
		checks = append(checks, check)
	}
	return checks
}
