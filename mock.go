package targetsync

import (
	"context"
	"fmt"
	"sync"
)

type mockLocker struct{}

func (m *mockLocker) Lock(context.Context, *LockOptions) (<-chan bool, error) {
	ch := make(chan bool, 1)
	ch <- true
	return ch, nil
}

func newmockSource() *mockSource {
	return &mockSource{
		ch: make(chan []*Target),
	}
}

type mockSource struct {
	ch chan []*Target
}

func (m *mockSource) Subscribe(context.Context) (chan []*Target, error) {
	return m.ch, nil
}

func newmockDestination() *mockDestination {
	return &mockDestination{
		targets: make([]*Target, 0),
	}
}

type mockDestination struct {
	targets []*Target
	l       sync.RWMutex
}

// GetTargets returns the current set of targets at the destination
func (m *mockDestination) GetTargets(context.Context) ([]*Target, error) {
	m.l.RLock()
	defer m.l.RUnlock()
	return m.targets, nil
}

// AddTargets simply adds the targets described
func (m *mockDestination) AddTargets(_ context.Context, tgts []*Target) error {
	m.l.Lock()
	defer m.l.Unlock()
	m.targets = append(m.targets, tgts...)
	return nil
}

// RemoveTargets simply removes the targets described
func (m *mockDestination) RemoveTargets(_ context.Context, tgts []*Target) error {
	m.l.Lock()
	defer m.l.Unlock()

	for _, tgt := range tgts {
		foundIdx := -1
		for idx, currentTgt := range m.targets {
			if tgt.Key() == currentTgt.Key() {
				foundIdx = idx
				break
			}
		}
		if foundIdx < 0 {
			return fmt.Errorf("Unable to remove target we don't have")
		}
		m.targets = append(m.targets[:foundIdx], m.targets[foundIdx+1:]...)
	}
	return nil
}
