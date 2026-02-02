package checks

import (
	"sync"
)

// DefaultRegistry is the global check registry.
// All checks should register themselves with this registry.
var DefaultRegistry = NewRegistry()

// Registry maintains a mapping of check names to Check instances.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]Check
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		checks: make(map[string]Check),
	}
}

// Register adds a check to the registry.
// If a check with the same name is already registered, it will be replaced.
func (r *Registry) Register(check Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[check.Name()] = check
}

// Get retrieves a check by name.
// Returns nil if the check is not registered.
func (r *Registry) Get(name string) Check {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.checks[name]
}

// Names returns a sorted list of all registered check names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.checks))
	for name := range r.checks {
		names = append(names, name)
	}
	return names
}

// All returns all registered checks.
func (r *Registry) All() []Check {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checks := make([]Check, 0, len(r.checks))
	for _, check := range r.checks {
		checks = append(checks, check)
	}
	return checks
}
