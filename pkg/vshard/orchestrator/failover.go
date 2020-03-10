package orchestrator

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/viciious/go-tarantool"

	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/vshard"
)

const (
	funcChangeMaster = "qumomf_change_master"
)

type Failover interface {
	Serve(stream AnalysisReadStream)
	Shutdown()
}

type swapMasterFailover struct {
	cluster vshard.Cluster
	elector quorum.Quorum
	stop    chan struct{}
}

func NewSwapMasterFailover(cluster vshard.Cluster, elector quorum.Quorum) Failover {
	return &swapMasterFailover{
		cluster: cluster,
		elector: elector,
		stop:    make(chan struct{}, 1),
	}
}

func (f *swapMasterFailover) Serve(stream AnalysisReadStream) {
	ctx := context.Background()

	go func() {
		for {
			select {
			case <-f.stop:
				return
			case analysis := <-stream:
				f.checkAndRecover(ctx, analysis)
			}
		}
	}()
}

func (f *swapMasterFailover) Shutdown() {
	f.stop <- struct{}{}
}

func (f *swapMasterFailover) checkAndRecover(ctx context.Context, analysis ReplicaSetAnalysis) {
	info := analysis.Info
	replicaSet := analysis.Set

	for _, replicaInfo := range info {
		uuid := replicaInfo.UUID
		switch replicaInfo.State {
		case vshard.DeadMaster:
			log.Info().Msgf("Found a dead master. Replica UUID: %s. Start rebuilding the shard topology.", uuid)
			f.cluster.StartRecovery()
			f.promoteFollowerToMaster(ctx, replicaSet, info)
			f.cluster.StopRecovery()
		case vshard.DeadSlave:
			log.Info().Msgf("Found a dead slave. Replica UUID: %s", uuid)
		case vshard.BadStorageInfo:
			log.Info().Msgf("Found a replica with unknown state. Replica UUID: %s", uuid)
		}
	}
}

func (f *swapMasterFailover) promoteFollowerToMaster(ctx context.Context, r vshard.ReplicaSet, info vshard.ReplicaSetInfo) {
	candidateUUID, err := f.elector.ChooseMaster(info)
	if err != nil {
		log.Info().Msgf("Failed to elect a new master: %s", err)
		return
	}

	log.Info().Msgf("New master is elected: %s. Going to update cluster configuration", candidateUUID)

	q := &tarantool.Call{
		Name: funcChangeMaster,
		Tuple: []interface{}{
			string(r.GetShardUUID()), string(r.GetMaster()), string(candidateUUID),
		},
	}

	// Update configuration on shards.
	for _, r := range f.cluster.GetReplicaSets() {
		for uuid, conn := range r.GetConnectors() {
			resp := conn.Exec(ctx, q)
			if resp.Error == nil {
				log.Info().Msgf("Configuration was updated on node %s", uuid)
			} else {
				log.Info().Msgf("Failed to update configuration on node %s: %s", uuid, resp.Error)
			}
		}
	}

	// Update configuration on routers.
	for uuid, conn := range f.cluster.GetRouterConnectors() {
		resp := conn.Exec(ctx, q)
		if resp.Error == nil {
			log.Info().Msgf("Configuration was updated on router %s", uuid)
		} else {
			log.Info().Msgf("Failed to update configuration on router %s: %s", uuid, resp.Error)
		}
	}

	r.SetMaster(candidateUUID)
}
