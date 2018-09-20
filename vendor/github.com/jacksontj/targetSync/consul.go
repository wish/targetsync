package targetsync

import (
	"context"

	consulApi "github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

// NewConsulSource returns a new ConsulSource
func NewConsulSource(cfg *ConsulConfig) (*ConsulSource, error) {
	consulCfg := consulApi.DefaultConfig()
	client, err := consulApi.NewClient(consulCfg)
	if err != nil {
		return nil, err
	}

	return &ConsulSource{
		cfg:          cfg,
		client:       client,
		healthClient: client.Health(),
	}, nil
}

// ConsulSource is an implementation for talkint to consul for both `TargetSource` and `Locker`
type ConsulSource struct {
	cfg          *ConsulConfig
	client       *consulApi.Client
	healthClient *consulApi.Health
}

// Lock to implement the Locker interface
func (s *ConsulSource) Lock(ctx context.Context, opts *LockOptions) (<-chan bool, error) {
	lock, err := s.client.LockOpts(&consulApi.LockOptions{
		Key:        opts.Key,
		SessionTTL: opts.TTL.String(),
	})
	if err != nil {
		return nil, err
	}

	lockedCh := make(chan bool, 1)

	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer close(lockedCh)
		defer cancel()

		stopCh := make(chan struct{})
		go func() {
			<-ctx.Done()
			close(stopCh)
		}()
		for {
			lockCh, err := lock.Lock(stopCh)
			if err != nil {
				logrus.Errorf("Error acquiring lock: %v", err)
				return
			}

			// We have the lock, start things up
			logrus.Infof("Lock acquired")
			lockedCh <- true

			select {
			case <-ctx.Done():
				return
			case <-lockCh:
				logrus.Infof("Lock lost")
				lockedCh <- false
			}
		}
	}()

	return lockedCh, nil
}

// Subscribe to implement the `TargetSource` interface
func (s *ConsulSource) Subscribe(ctx context.Context) (chan []*Target, error) {
	queryOpts := &consulApi.QueryOptions{
		WaitIndex: 0,
	}
	queryOpts = queryOpts.WithContext(ctx)

	// TODO: configurable size?
	ch := make(chan []*Target, 100)

	go func(ch chan []*Target) {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			services, meta, err := s.healthClient.Service(s.cfg.ServiceName, "", true, queryOpts)
			if err != nil {
				// TODO: sleep on failure with backoff etc.
				continue
			}

			// If there was a change
			if meta.LastIndex != queryOpts.WaitIndex {
				targets := make([]*Target, len(services))
				for i, entry := range services {
					addr := entry.Node.Address
					if entry.Service.Address != "" {
						addr = entry.Service.Address
					}
					targets[i] = &Target{
						IP:   addr,
						Port: entry.Service.Port,
					}
				}
				ch <- targets
			}

			queryOpts.WaitIndex = meta.LastIndex
		}
	}(ch)

	return ch, nil
}
