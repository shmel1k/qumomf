package orchestrator

import (
	"context"
	"log"
	"time"

	"github.com/viciious/go-tarantool"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

const (
	funcStorageInfo = "vshard.storage.info"
)

type Monitor interface {
	Serve() AnalysisReadStream
	Shutdown()
}

func NewMonitor(cfg Config, cluster vshard.Cluster) Monitor {
	return &storageMonitor{
		config:  cfg,
		cluster: cluster,
		stop:    make(chan struct{}, 1),
	}
}

type storageMonitor struct {
	config  Config
	cluster vshard.Cluster
	stop    chan struct{}
}

func (m *storageMonitor) analyzeReplicas(ctx context.Context, set vshard.ReplicaSet) ReplicaSetAnalysis {
	q := &tarantool.Call{
		Name: funcStorageInfo,
	}

	setInfo := vshard.ReplicaSetInfo{}
	masterUUID := set.GetMaster()

	for uuid, conn := range set.GetConnectors() {
		role := vshard.RoleFollow
		if uuid == masterUUID {
			role = vshard.RoleMaster
		}

		replicaInfo := vshard.ReplicaInfo{
			UUID:   uuid,
			Role:   role,
			Status: vshard.NoProblem,
		}

		infoResponse := conn.Exec(ctx, q)
		if infoResponse.Error == nil {
			info, err := parseStorageInfo(infoResponse.Data)
			if err == nil {
				replicaInfo.Lag = info.Replication.Lag
				replicaInfo.Alerts = info.Alerts

				if len(info.Alerts) > 0 {
					replicaInfo.Status = vshard.HasActiveAlerts
				}
			} else {
				log.Println(err)
				replicaInfo.Status = vshard.BadStorageInfo
			}
		} else {
			log.Println(infoResponse.Error)

			switch role {
			case vshard.RoleMaster:
				replicaInfo.Status = vshard.DeadMaster
			case vshard.RoleFollow:
				replicaInfo.Status = vshard.DeadSlave
			}
		}

		setInfo = append(setInfo, replicaInfo)
		log.Println(replicaInfo)
	}

	return ReplicaSetAnalysis{
		Set:  set,
		Info: setInfo,
	}
}

func (m *storageMonitor) serveReplicaSet(r vshard.ReplicaSet, stream AnalysisWriteStream) {
	tick := time.NewTicker(m.config.CheckTimeout)
	defer tick.Stop()

	ctx := context.Background()

	for {
		select {
		case <-m.stop:
			return
		case <-tick.C:
			if !m.cluster.HasActiveRecovery() {
				stream <- m.analyzeReplicas(ctx, r)
			}
		}
	}
}

func (m *storageMonitor) Serve() AnalysisReadStream {
	stream := NewAnalysisStream()

	go func() {
		for _, v := range m.cluster.GetReplicaSets() {
			go func(set vshard.ReplicaSet) {
				m.serveReplicaSet(set, stream)
			}(v)
		}
	}()

	return stream
}

func (m *storageMonitor) Shutdown() {
	m.stop <- struct{}{}
}
