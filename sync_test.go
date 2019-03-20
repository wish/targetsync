package targetsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func equalTargets(a, b []*Target) error {
	aMap := make(map[string]*Target)
	for _, target := range a {
		aMap[target.Key()] = target
	}
	bMap := make(map[string]*Target)
	for _, target := range b {
		bMap[target.Key()] = target
	}

	if len(aMap) != len(bMap) {
		return fmt.Errorf("Mismatch in len")
	}

	for k, _ := range aMap {
		if _, ok := bMap[k]; !ok {
			return fmt.Errorf("b is missing %s", k)
		}
	}
	return nil
}

func TestSyncer(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	cfg := &SyncConfig{
		LockOptions: LockOptions{
			Key: "a",
			TTL: time.Second,
		},
		RemoveDelay: time.Second,
	}

	src := newmockSource()
	dst := newmockDestination()
	syncer := &Syncer{
		Config: cfg,
		Locker: &mockLocker{},
		Src:    src,
		Dst:    dst,
	}

	go syncer.Run(context.TODO())

	targets := []*Target{
		&Target{IP: "1"},
		&Target{IP: "2"},
	}

	// set targets
	src.ch <- targets
	time.Sleep(time.Second)

	// check that they match
	tgts, _ := dst.GetTargets(nil)
	if err := equalTargets(targets, tgts); err != nil {
		t.Fatalf("Mismatch in targets err=%v expected=%+v actual=%+v", err, targets, tgts)
	}

	time.Sleep(time.Second * 2)

	removed := []*Target{targets[0]}
	src.ch <- removed
	time.Sleep(time.Second * 2)

	// check that they match
	tgts, _ = dst.GetTargets(nil)
	if err := equalTargets(removed, tgts); err != nil {
		t.Fatalf("Mismatch in targets err=%v expected=%+v actual=%+v", err, removed, tgts)
	}

	empty := []*Target{}
	src.ch <- empty
	time.Sleep(time.Second * 2)

	// check that they match
	tgts, _ = dst.GetTargets(nil)
	if err := equalTargets(empty, tgts); err != nil {
		t.Fatalf("Mismatch in targets err=%v expected=%+v actual=%+v", err, empty, tgts)
	}
}
