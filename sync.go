package targetsync

import (
	"context"
	"fmt"
	"time"

	"github.com/jacksontj/lane"
	"github.com/sirupsen/logrus"
)

// Syncer is the struct that uses the various interfaces to actually do the sync
// TODO: metrics
type Syncer struct {
	Config    *SyncConfig
	LocalAddr string
	Locker    Locker
	Src       TargetSource
	Dst       TargetDestination
	Started   bool
}

// syncSelf simply syncs the LocalAddr from the souce to the target
func (s *Syncer) syncSelf(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	logrus.Infof("Local Addr %s -- waiting until added to target", s.LocalAddr)
	srcCh, err := s.Src.Subscribe(ctx)
	if err != nil {
		return err
	}

	// Now we wait until our IP shows up in the source data, once it does
	// we add ourselves to the target
	for {
		logrus.Debugf("Waiting for targets from source")
		var srcTargets []*Target
		select {
		case <-ctx.Done():
			return ctx.Err()
		case srcTargets = <-srcCh:
		}
		logrus.Debugf("Received targets from source: %+#v", srcTargets)

		for _, target := range srcTargets {
			if target.IP == s.LocalAddr {
				// try adding ourselves
				if err := s.Dst.AddTargets(ctx, []*Target{target}); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

// Run is the main method for the syncer. This is responsible for calling
// runLeader when the lock is held
func (s *Syncer) Run(ctx context.Context) error {
	// add ourselves if a LocalAddr was defined
	if s.LocalAddr != "" {
		if err := s.syncSelf(ctx); err != nil {
			return err
		}
	}

	s.Started = true
	logrus.Debugf("Syncer creating lock: %v", s.Config.LockOptions)
	electedCh, err := s.Locker.Lock(ctx, &s.Config.LockOptions)
	if err != nil {
		return err
	}

	var leaderCtx context.Context
	var leaderCtxCancel context.CancelFunc

	for {
		select {
		case <-ctx.Done():
			if leaderCtxCancel != nil {
				leaderCtxCancel()
			}
			return ctx.Err()
		case elected, ok := <-electedCh:
			if !ok {
				return fmt.Errorf("Lock channel closed")
			}
			if elected {
				leaderCtx, leaderCtxCancel = context.WithCancel(ctx)
				logrus.Infof("Lock acquired, starting leader actions")
				go s.runLeader(leaderCtx)
			} else {
				logrus.Infof("Lock lost, stopping leader actions")
				if leaderCtxCancel != nil {
					leaderCtxCancel()
				}
			}
		}
	}
}

// bgRemove is a background goroutine responsible for removing targets from the destination
// this exists to allow for a `RemoveDelay` on the removal of targets from the destination
// to avoid issues where a target is "flapping" in the source
func (s *Syncer) bgRemove(ctx context.Context, removeCh chan *Target, addCh chan *Target) {
	itemMap := make(map[string]*lane.Item)
	q := lane.NewPQueue(lane.MINPQ)

	defaultDuration := time.Hour

	t := time.NewTimer(defaultDuration)
	for {
		select {
		case <-ctx.Done():
			return
		case toRemove, ok := <-removeCh:
			if !ok {
				continue
			}

			// This means the target is already scheduled for removal
			if _, ok := itemMap[toRemove.Key()]; ok {
				continue
			}

			logrus.Debugf("Scheduling target for removal from destination in %v: %v", s.Config.RemoveDelay, toRemove)
			now := time.Now()
			removeUnixTime := now.Add(s.Config.RemoveDelay).Unix()
			if headItem, headAt := q.Head(); headItem == nil || removeUnixTime < headAt {
				if !t.Stop() {
					select {
					case <-t.C:
					default:
					}
				}
				t.Reset(s.Config.RemoveDelay)
			}
			itemMap[toRemove.Key()] = q.Push(toRemove, removeUnixTime)
		case toAdd, ok := <-addCh:
			if !ok {
				continue
			}
			key := toAdd.Key()
			if item, ok := itemMap[key]; ok {
				logrus.Debugf("Removing target from removal queue as it was re-added: %v", toAdd)
				q.Remove(item)
				delete(itemMap, key)
			}
		case <-t.C:
			// Check if there is an item at head, and if the time is past then
			// do the removal
			headItem, headUnixTime := q.Head()
			logrus.Debugf("Processing target removal: %v", headItem)
			now := time.Now()
			nowUnix := now.Unix()

		DELETE_LOOP:
			for headItem != nil {
				// If we where woken before something is ready, just reschedule
				if headUnixTime > nowUnix {
					break DELETE_LOOP
				} else {
					target := headItem.(*Target)
					if err := s.Dst.RemoveTargets(ctx, []*Target{target}); err == nil {
						logrus.Debugf("Target removal successful: %v", target)
						q.Pop()
						delete(itemMap, target.Key())
					} else {
						logrus.Errorf("Target removal unsuccessful %v: %v", target, err)
						break DELETE_LOOP
					}
				}
				headItem, headUnixTime = q.Head()
			}
			// If there is still an item in the queue, reset the timer
			if headItem != nil {
				d := time.Unix(headUnixTime, 0).Sub(now)
				if !t.Stop() {
					select {
					case <-t.C:
					default:
					}
				}
				t.Reset(d)
			}
		}
	}
}

// runLeader does the actual syncing from source to destination. This is called
// after the leader election has been done, there should only be one of these per
// unique destination running globally
func (s *Syncer) runLeader(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	removeCh := make(chan *Target, 100)
	addCh := make(chan *Target, 100)
	defer close(removeCh)
	defer close(addCh)
	go s.bgRemove(ctx, removeCh, addCh)

	// get state from source
	srcCh, err := s.Src.Subscribe(ctx)
	if err != nil {
		return err
	}

	// Wait for an update, if we get one sync it
	for {
		logrus.Debugf("Waiting for targets from source")
		var srcTargets []*Target
		select {
		case <-ctx.Done():
			return ctx.Err()
		case srcTargets = <-srcCh:
		}
		logrus.Debugf("Received targets from source: %+#v", srcTargets)

		// get current ones from dst
		dstTargets, err := s.Dst.GetTargets(ctx)
		if err != nil {
			return err
		}
		logrus.Debugf("Fetched targets from destination: %+#v", dstTargets)

		// TODO: compare ports and do something with them
		srcMap := make(map[string]*Target)
		for _, target := range srcTargets {
			srcMap[target.IP] = target
		}
		dstMap := make(map[string]*Target)
		for _, target := range dstTargets {
			dstMap[target.IP] = target
		}

		// Add hosts first
		hostsToAdd := make([]*Target, 0)
		for ip, target := range srcMap {
			// We want to ensure that any target we think should be alive isn't
			// in the removal queue
			addCh <- target

			if _, ok := dstMap[ip]; !ok {
				hostsToAdd = append(hostsToAdd, target)
			}
		}
		if len(hostsToAdd) > 0 {
			logrus.Debugf("Adding targets to destination: %v", hostsToAdd)
			if err := s.Dst.AddTargets(ctx, hostsToAdd); err != nil {
				return err
			}
		}

		// Remove hosts last
		for ip, target := range dstMap {
			if _, ok := srcMap[ip]; !ok {
				removeCh <- target
			}
		}
	}
}
