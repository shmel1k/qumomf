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
	cluster *vshard.Cluster
	elector quorum.Quorum
	stop    chan struct{}
}

func NewSwapMasterFailover(cluster *vshard.Cluster, elector quorum.Quorum) Failover {
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
				if f.shouldBeAnalysisChecked() {
					f.checkAndRecover(ctx, analysis)
				}
			}
		}
	}()
}

func (f *swapMasterFailover) Shutdown() {
	f.stop <- struct{}{}
}

func (f *swapMasterFailover) shouldBeAnalysisChecked() bool {
	if f.cluster.ReadOnly() {
		log.Debug().Msgf("Cluster '%s' is readonly. Skip check and recovery step for all shards.", f.cluster.Name)
		return false
	}
	if f.cluster.HasActiveRecovery() {
		log.Debug().Msgf("Cluster '%s' has active recovery. Skip check and recovery step for all shards.", f.cluster.Name)
		return false
	}
	return true
}

func (f *swapMasterFailover) checkAndRecover(ctx context.Context, analysis *ReplicationAnalysis) {
	set := analysis.Set

	switch analysis.State {
	case DeadMaster:
		f.cluster.StartRecovery()
		log.Info().Msgf("Master cannot be reached by qumomf. Will run failover. ReplicaSet snapshot: %s", set)
		f.promoteFollowerToMaster(ctx, set)
		f.cluster.StopRecovery()
	case DeadMasterAndSomeFollowers:
		f.cluster.StartRecovery()
		log.Info().Msgf("Master cannot be reached by qumomf and some of its followers are unreachable. Will run failover. ReplicaSet snapshot: %s", set)
		f.promoteFollowerToMaster(ctx, set)
		f.cluster.StopRecovery()
	case DeadMasterAndFollowers:
		log.Info().Msgf("Master cannot be reached by qumomf and none of its followers is replicating. No actions will be applied. ReplicaSet snapshot: %s", set)
	case AllMasterFollowersNotReplicating:
		log.Info().Msgf("Master is reachable but none of its replicas is replicating. No actions will be applied. ReplicaSet snapshot: %s", set)
	case DeadMasterWithoutFollowers:
		log.Info().Msgf("Master cannot be reached by qumomf and has no followers. No actions will be applied. ReplicaSet snapshot: %s", set)
	case NoProblem:
		// Nothing to do, everything is OK.
	}
}

func (f *swapMasterFailover) promoteFollowerToMaster(ctx context.Context, badSet vshard.ReplicaSet) {
	candidateUUID, err := f.elector.ChooseMaster(badSet)
	if err != nil {
		log.Error().Msgf("Failed to elect a new master: %s", err)
		return
	}

	log.Info().Msgf("New master is elected: %s. Going to update cluster configuration", candidateUUID)

	q := &tarantool.Call{
		Name: funcChangeMaster,
		Tuple: []interface{}{
			string(badSet.UUID), string(badSet.MasterUUID), string(candidateUUID),
		},
	}

	// Update configuration on replica sets.
	for _, set := range f.cluster.ReplicaSets() {
		for i := range set.Instances {
			inst := &set.Instances[i]
			conn := f.cluster.Pool.Get(inst.URI, string(inst.UUID))
			resp := conn.Exec(ctx, q)
			if resp.Error == nil {
				log.Info().Msgf("Configuration was updated on node '%s'", inst.UUID)
			} else {
				log.Error().Msgf("Failed to update configuration on node '%s': %s", inst.UUID, resp.Error)
			}
		}
	}

	// Update configuration on routers.
	for _, r := range f.cluster.Routers() {
		conn := f.cluster.Pool.Get(r.URI, string(r.UUID))
		resp := conn.Exec(ctx, q)
		if resp.Error == nil {
			log.Info().Msgf("Configuration was updated on router '%s'", r.UUID)
		} else {
			log.Error().Msgf("Failed to update configuration on router '%s': %s", r.UUID, resp.Error)
		}
	}
}
