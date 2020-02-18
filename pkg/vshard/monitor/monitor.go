package monitor

import (
	"context"
	"log"
	"time"

	"github.com/shmel1k/qumomf/pkg/vshard"
	"github.com/viciious/go-tarantool"
)

const (
	funcStorageInfo = "vshard.storage.info"
)

type Monitor interface {
	Serve() <-chan error
}

func New(cfg Config, cluster vshard.Cluster) Monitor {
	return &storageMonitor{
		config:  cfg,
		cluster: cluster,
	}
}

type storageMonitor struct {
	config  Config
	cluster vshard.Cluster
	stop    chan struct{}
}

func (m *storageMonitor) checkReplicas(ctx context.Context, r vshard.Replicaset) error {
	q := &tarantool.Call{
		Name: funcStorageInfo,
	}

	for _, set := range r.GetReplicas() {
		infoResponse := set.Exec(ctx, q)
		if infoResponse.Error != nil {
			return infoResponse.Error
		}

		info, err := parseStorageInfo(infoResponse.Data)
		if err != nil {
			log.Println(err)
		}
		log.Println(info)
	}
	return nil
}

func (m *storageMonitor) serveReplicaSet(r vshard.Replicaset) error {
	tick := time.NewTicker(m.config.CheckTimeout)
	defer tick.Stop()

	ctx := context.Background()

	for {
		select {
		case <-m.stop:
			return nil
		case <-tick.C:
		}

		err := m.checkReplicas(ctx, r)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (m *storageMonitor) Serve() <-chan error {
	errs := make(chan error)

	go func() {
		for _, v := range m.cluster.GetReplicasets() {
			go func(set vshard.Replicaset) {
				if err := m.serveReplicaSet(set); err != nil {
					errs <- err
				}
			}(v)
		}
	}()
	return errs
}
