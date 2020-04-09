package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/viciious/go-tarantool"

	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/util"
	"github.com/shmel1k/qumomf/pkg/vshard"
)

const (
	funcChangeMaster = "qumomf_change_master"
)

const (
	cleanupPeriod = 1 * time.Minute
)

type Failover interface {
	Serve(stream AnalysisReadStream)
	Shutdown()
}

type swapMasterFailover struct {
	cluster *vshard.Cluster
	elector quorum.Quorum

	blockers    []*BlockedRecovery
	blockerSync sync.RWMutex
	blockerTTL  time.Duration

	stop chan struct{}
}

func NewSwapMasterFailover(cluster *vshard.Cluster, cfg FailoverConfig) Failover {
	return &swapMasterFailover{
		cluster:    cluster,
		elector:    cfg.Elector,
		blockers:   make([]*BlockedRecovery, 0),
		blockerTTL: cfg.ReplicaSetRecoveryBlockTime,
		stop:       make(chan struct{}, 1),
	}
}

func (f *swapMasterFailover) Serve(stream AnalysisReadStream) {
	ctx := context.Background()

	cleanupTick := time.NewTicker(cleanupPeriod)
	defer cleanupTick.Stop()

	go func() {
		for {
			select {
			case <-f.stop:
				return
			case <-cleanupTick.C:
				f.cleanup(false)
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
	log.Debug().Msgf("checkAndRecover: %s", *analysis)
	set := analysis.Set

	switch analysis.State {
	case NoProblem:
		// Nothing to do, everything is OK.
	case DeadMaster:
		f.cluster.StartRecovery()
		log.Info().Msgf("Master cannot be reached by qumomf. Will run failover. ReplicaSet snapshot: %s", set)
		recv := f.promoteFollowerToMaster(ctx, analysis)
		if recv != nil {
			f.registryRecovery(recv)
			log.Info().Msgf("Finished recovery: %s", *recv)
			log.Info().Msgf("Run a force discovery after the recovery on ReplicaSet '%s'", set.UUID)
			f.cluster.Discover()
		}
		f.cluster.StopRecovery()
	case DeadMasterAndSomeFollowers:
		f.cluster.StartRecovery()
		log.Info().Msgf("Master cannot be reached by qumomf and some of its followers are unreachable. Will run failover. ReplicaSet snapshot: %s", set)
		recv := f.promoteFollowerToMaster(ctx, analysis)
		if recv != nil {
			f.registryRecovery(recv)
			log.Info().Msgf("Recovery status: %s", *recv)
			log.Info().Msgf("Run a force discovery after the recovery on ReplicaSet '%s'", set.UUID)
			f.cluster.Discover()
		}
		f.cluster.StopRecovery()
	case DeadMasterAndFollowers:
		log.Info().Msgf("Master cannot be reached by qumomf and none of its followers is replicating. No actions will be applied. ReplicaSet snapshot: %s", set)
	case AllMasterFollowersNotReplicating:
		log.Info().Msgf("Master is reachable but none of its replicas is replicating. No actions will be applied. ReplicaSet snapshot: %s", set)
	case DeadMasterWithoutFollowers:
		log.Info().Msgf("Master cannot be reached by qumomf and has no followers. No actions will be applied. ReplicaSet snapshot: %s", set)
	case NetworkProblems:
		log.Info().Msgf("Master cannot be reached by qumomf but some followers are still replicating. It might be a network problem, no actions will be applied. ReplicaSet snapshot: %s", set)
	}
}

func (f *swapMasterFailover) promoteFollowerToMaster(ctx context.Context, analysis *ReplicationAnalysis) *Recovery {
	badSet := analysis.Set

	if f.hasBlockedRecovery(badSet.UUID) {
		log.Warn().Msgf("ReplicaSet %s has been recovered recently so new failover is blocked", badSet.UUID)
		return nil
	}

	recv := NewRecovery(analysis)

	candidateUUID, err := f.elector.ChooseMaster(badSet)
	if err != nil {
		log.Error().Msgf("Failed to elect a new master: %s", err)

		recv.IsSuccessful = false
		recv.EndTimestamp = util.Timestamp()

		return recv
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

			if inst.UUID == candidateUUID {
				err = f.setReadOnly(ctx, conn, false)
				if err == nil {
					log.Info().Msgf("Applied read_only = false to node '%s'", inst.UUID)
				} else {
					log.Error().Msgf("Failed to disable readonly on node '%s': %s", inst.UUID, err)
				}
			}

			// Just try
			if inst.UUID == badSet.MasterUUID {
				err = f.setReadOnly(ctx, conn, true)
				if err == nil {
					log.Info().Msgf("Applied read_only = true to node '%s'", inst.UUID)
				} else {
					log.Error().Msgf("Failed to enable readonly on node '%s': %s", inst.UUID, err)
				}
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

	recv.IsSuccessful = true
	recv.SuccessorUUID = candidateUUID
	recv.EndTimestamp = util.Timestamp()

	return recv
}

func (f *swapMasterFailover) setReadOnly(ctx context.Context, conn *vshard.Connector, ro bool) error {
	call := &tarantool.Eval{
		Expression: `
			local arg = {...}
			box.cfg({
        		read_only = arg[1],
    		})
		`,
		Tuple: []interface{}{ro},
	}

	resp := conn.Exec(ctx, call)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (f *swapMasterFailover) registryRecovery(r *Recovery) {
	blocker := NewBlockedRecovery(r, f.blockerTTL)

	f.blockerSync.Lock()
	f.blockers = append(f.blockers, blocker)
	f.blockerSync.Unlock()
}

func (f *swapMasterFailover) hasBlockedRecovery(uuid vshard.ReplicaSetUUID) bool {
	f.blockerSync.RLock()
	defer f.blockerSync.RUnlock()

	for _, b := range f.blockers {
		if b.Recovery.SetUUID == uuid && !b.Expired() {
			return true
		}
	}

	return false
}

func (f *swapMasterFailover) cleanup(force bool) {
	// It is not a frequent operation, so do not
	// see any reason to optimize this place.

	f.blockerSync.RLock()
	if len(f.blockers) == 0 {
		f.blockerSync.RUnlock()
		return
	}

	alive := make([]*BlockedRecovery, 0)
	if !force {
		for _, b := range f.blockers {
			if !b.Expired() {
				alive = append(alive, b)
			}
		}
	}
	f.blockerSync.RUnlock()

	f.blockerSync.Lock()
	f.blockers = alive
	f.blockerSync.Unlock()
}
