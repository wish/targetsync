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
		return fmt.Errorf("Mismatch in len a=%d b=%d", len(aMap), len(bMap))
	}

	for k := range aMap {
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
		{IP: "1"},
		{IP: "2"},
	}
	target := []*Target{targets[0]}
	empty := []*Target{}

	// set targets
	src.ch <- targets
	time.Sleep(time.Second)

	// check that they match
	tgts, _ := dst.GetTargets(nil)
	if err := equalTargets(targets, tgts); err != nil {
		t.Fatalf("Mismatch in targets err=%v expected=%+v actual=%+v", err, targets, tgts)
	}

	time.Sleep(time.Second * 2)

	src.ch <- target
	time.Sleep(time.Second * 2)

	// check that they match
	tgts, _ = dst.GetTargets(nil)
	if err := equalTargets(target, tgts); err != nil {
		t.Fatalf("Mismatch in targets err=%v expected=%+v actual=%+v", err, target, tgts)
	}

	src.ch <- empty
	time.Sleep(time.Second * 2)

	// check that they match
	tgts, _ = dst.GetTargets(nil)
	if err := equalTargets(empty, tgts); err != nil {
		t.Fatalf("Mismatch in targets err=%v expected=%+v actual=%+v", err, empty, tgts)
	}
}

// TestSyncer_Races is specifically checking for issues where removals and adds are racing with eachother
func TestSyncer_Races(t *testing.T) {
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

	target := []*Target{{IP: "1"}}
	empty := []*Target{}

	// Add 1
	src.ch <- target
	time.Sleep(time.Second * 2)

	// remove it twice
	src.ch <- empty
	time.Sleep(time.Millisecond * 300)
	src.ch <- empty
	time.Sleep(time.Millisecond * 300)

	// add 1
	// sleep 1s
	src.ch <- target
	time.Sleep(time.Second * 2)

	// check
	tgts, _ := dst.GetTargets(nil)
	if err := equalTargets(target, tgts); err != nil {
		t.Fatalf("Mismatch in targets err=%v expected=%+v actual=%+v", err, target, tgts)
	}
}
