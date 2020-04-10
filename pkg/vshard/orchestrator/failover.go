package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
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

	stop   chan struct{}
	logger zerolog.Logger
}

func NewSwapMasterFailover(cluster *vshard.Cluster, cfg FailoverConfig) Failover {
	return &swapMasterFailover{
		logger:     cfg.Logger,
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
		f.logger.Info().Msgf("Readonly cluster: skip check and recovery step for all shards")
		return false
	}
	if f.cluster.HasActiveRecovery() {
		f.logger.Info().Msgf("Cluster has active recovery: skip check and recovery step for all shards")
		return false
	}
	return true
}

func (f *swapMasterFailover) checkAndRecover(ctx context.Context, analysis *ReplicationAnalysis) {
	f.logger.Info().Msgf("checkAndRecover: %s", *analysis)
	set := analysis.Set

	switch analysis.State {
	case NoProblem:
		// Nothing to do, everything is OK.
	case DeadMaster:
		f.cluster.StartRecovery()
		f.logger.Info().Msgf("Master cannot be reached by qumomf. Will run failover. ReplicaSet snapshot: %s", set)
		recv := f.promoteFollowerToMaster(ctx, analysis)
		if recv != nil {
			f.registryRecovery(recv)
			f.logger.Info().Msgf("Finished recovery: %s", *recv)
			f.logger.Info().Msgf("Run a force discovery after the recovery on ReplicaSet '%s'", set.UUID)
			f.cluster.Discover()
		}
		f.cluster.StopRecovery()
	case DeadMasterAndSomeFollowers:
		f.cluster.StartRecovery()
		f.logger.Info().Msgf("Master cannot be reached by qumomf and some of its followers are unreachable. Will run failover. ReplicaSet snapshot: %s", set)
		recv := f.promoteFollowerToMaster(ctx, analysis)
		if recv != nil {
			f.registryRecovery(recv)
			f.logger.Info().Msgf("Recovery status: %s", *recv)
			f.logger.Info().Msgf("Run a force discovery after the recovery on ReplicaSet '%s'", set.UUID)
			f.cluster.Discover()
		}
		f.cluster.StopRecovery()
	case DeadMasterAndFollowers:
		f.logger.Info().Msgf("Master cannot be reached by qumomf and none of its followers is replicating. No actions will be applied. ReplicaSet snapshot: %s", set)
	case AllMasterFollowersNotReplicating:
		f.logger.Info().Msgf("Master is reachable but none of its replicas is replicating. No actions will be applied. ReplicaSet snapshot: %s", set)
	case DeadMasterWithoutFollowers:
		f.logger.Info().Msgf("Master cannot be reached by qumomf and has no followers. No actions will be applied. ReplicaSet snapshot: %s", set)
	case NetworkProblems:
		f.logger.Info().Msgf("Master cannot be reached by qumomf but some followers are still replicating. It might be a network problem, no actions will be applied. ReplicaSet snapshot: %s", set)
	}
}

func (f *swapMasterFailover) promoteFollowerToMaster(ctx context.Context, analysis *ReplicationAnalysis) *Recovery {
	badSet := analysis.Set
	logger := f.logger.With().Str("ReplicaSet", string(badSet.UUID)).Logger()

	if f.hasBlockedRecovery(badSet.UUID) {
		logger.Warn().Msg("ReplicaSet has been recovered recently so new failover is blocked")
		return nil
	}

	recv := NewRecovery(analysis)

	candidateUUID, err := f.elector.ChooseMaster(badSet)
	if err != nil {
		logger.Err(err).Msg("Failed to elect a new master")

		recv.IsSuccessful = false
		recv.EndTimestamp = util.Timestamp()

		return recv
	}

	logger.Info().Msgf("New master is elected: %s. Going to update cluster configuration", candidateUUID)

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
				logger.Info().Msgf("Configuration was updated on node '%s'", inst.UUID)
			} else {
				logger.Err(resp.Error).Msgf("Failed to update configuration on node '%s'", inst.UUID)
			}

			if inst.UUID == candidateUUID {
				err = f.setReadOnly(ctx, conn, false)
				if err == nil {
					logger.Info().Msgf("Applied 'read_only=false' to node '%s'", inst.UUID)
				} else {
					logger.Err(err).Msgf("Failed to apply 'read_only=false' on node '%s'", inst.UUID)
				}
			}

			// Just try
			if inst.UUID == badSet.MasterUUID {
				err = f.setReadOnly(ctx, conn, true)
				if err == nil {
					logger.Info().Msgf("Applied 'read_only=true' to node '%s'", inst.UUID)
				} else {
					logger.Err(err).Msgf("Failed to apply 'read_only=true' on node '%s'", inst.UUID)
				}
			}
		}
	}

	// Update configuration on routers.
	for _, r := range f.cluster.Routers() {
		conn := f.cluster.Pool.Get(r.URI, string(r.UUID))
		resp := conn.Exec(ctx, q)
		if resp.Error == nil {
			logger.Info().Msgf("Configuration was updated on router '%s'", r.UUID)
		} else {
			logger.Err(resp.Error).Msgf("Failed to update configuration on router '%s'", r.UUID)
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
