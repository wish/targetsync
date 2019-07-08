package targetsync

import (
	"context"
	"fmt"
	"time"
)

// Target represents a single IP+Port pair
type Target struct {
	IP   string
	Port int
}

// Key returns a unique key identifying this specific target
func (t *Target) Key() string {
	return fmt.Sprintf("%s:%d", t.IP, t.Port)
}

// TargetSource is an interface for getting targets for a given config
// TODO: plugin etc.
type TargetSource interface {
	Subscribe(context.Context) (chan []*Target, error)
}

// TargetDestination is a place to apply targets to (e.g. TargetGroup)
type TargetDestination interface {
	// GetTargets returns the current set of targets at the destination
	GetTargets(context.Context) ([]*Target, error)
	// AddTargets simply adds the targets described
	AddTargets(context.Context, []*Target) error
	// RemoveTargets simply removes the targets described
	RemoveTargets(context.Context, []*Target) error
}

// LockOptions holds the options for locking/leader-election
type LockOptions struct {
	Key string        `yaml:"key"`
	TTL time.Duration `yaml:"ttl"`
}

// Locker is an interface for locking/leader-election
type Locker interface {
	// Lock will acquire the lock defined in `LockOptions` and return a channel
	// which will respond with whether we are the leader
	Lock(context.Context, *LockOptions) (<-chan bool, error)
}

type TargetSourceLocker interface {
	Locker
	TargetSource
}
