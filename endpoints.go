package targetsync

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type K8sEndpointsSource struct {
	clientset       *kubernetes.Clientset
	name, namespace string
	port            int
}

func NewK8sEndpointsSource(cfg *K8sEndpointsConfig) (*K8sEndpointsSource, error) {
	var config *rest.Config
	var err error
	if cfg.K8sConfig.InCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", cfg.K8sConfig.KubeConfigPath)
		if err != nil {
			return nil, err
		}
	}

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sEndpointsSource{
		clientset: c,
		name:      cfg.Name,
		namespace: cfg.Namespace,
		port:      cfg.Port,
	}, nil
}

func (s *K8sEndpointsSource) Subscribe(ctx context.Context) (chan []*Target, error) {
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
			targets := []*Target{}

			ends, err := s.clientset.CoreV1().Endpoints(s.namespace).Get(s.name, metav1.GetOptions{})
			if err != nil {
				continue
			}

			for _, subset := range ends.Subsets {
				for _, addr := range subset.Addresses {
					targets = append(targets, &Target{
						IP:   addr.IP,
						Port: s.port,
					})
				}
			}
			ch <- targets
		}
	}(ch)

	return ch, nil
}

func (s *K8sEndpointsSource) Lock(ctx context.Context, opts *LockOptions) (<-chan bool, error) {
	leaseLockName := opts.Key
	leaseLockNamespace := s.namespace

	lock := &resourcelock.ConfigMapLock{
		ConfigMapMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: leaseLockNamespace,
		},
		Client: s.clientset.CoreV1(),
	}

	lockedCh := make(chan bool, 1)

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   opts.TTL,
		// TODO: Make configrable
		RenewDeadline: 5 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logrus.Infof("Lock acquired")
				lockedCh <- true
			},
			OnStoppedLeading: func() {
				logrus.Infof("Lock lost")
				lockedCh <- false
			},
			OnNewLeader: func(identity string) {},
		},
	})

	return lockedCh, nil
}
