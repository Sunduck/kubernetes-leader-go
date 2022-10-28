package main

import (
	"context"
	"log"
	"os"
	"sync/atomic"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func main() {
	var leader int32 // 1 if current pod is a leader, 0 if not

	// value populated in deploy.yaml manifest
	podName := os.Getenv("POD_NAME")

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Print(err.Error())
		return
	}

	clusterClientSet, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Print(err.Error())
		return
	}

	// context will drop current pod from election process on cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// starting leader election routine
	go func() {
		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock: &resourcelock.LeaseLock{
				LeaseMeta: v1.ObjectMeta{
					// two values must be same on all pods - initialization of
					// variable that will be locked by leader
					Name:      "lease-name",
					Namespace: "lease-namespace",
				},
				Client: clusterClientSet.CoordinationV1(),
				LockConfig: resourcelock.ResourceLockConfig{
					Identity: podName,
				},
			},
			ReleaseOnCancel: true,
			LeaseDuration:   15 * time.Second,
			RenewDeadline:   10 * time.Second,
			RetryPeriod:     2 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					atomic.StoreInt32(&leader, 1)
					log.Printf("Pod %s now a leader", podName)
					DoLeaderStuff(ctx)
				},
				OnStoppedLeading: func() {
					atomic.StoreInt32(&leader, 0)
					log.Printf("Pod %s not a leader anymore", podName)
				},
				OnNewLeader: func(id string) {
					if id == podName {
						// current pod is a leader
						return
					}
					log.Printf("Pod %s is a new leader", podName)
				},
			},
		})
	}()
}

func DoLeaderStuff(ctx context.Context) {
}
