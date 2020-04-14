package orchestrator

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/viciious/go-tarantool"

	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/util"
	"github.com/shmel1k/qumomf/pkg/vshard"
)

const (
	cleanupPeriod = 1 * time.Minute
)

type recoveryRole string

const (
	roleSuccessor recoveryRole = "successor"
	roleReplica   recoveryRole = "replica"
	roleFailed    recoveryRole = "failed"
	roleRouter    recoveryRole = "router"
)

type Failover interface {
	Serve(stream AnalysisReadStream)
	Shutdown()
}

type promoteFailover struct {
	cluster *vshard.Cluster
	elector quorum.Quorum

	blockers    []*BlockedRecovery
	blockerSync sync.RWMutex
	blockerTTL  time.Duration

	stop   chan struct{}
	logger zerolog.Logger
}

func NewPromoteFailover(cluster *vshard.Cluster, cfg FailoverConfig) Failover {
	return &promoteFailover{
		logger:     cfg.Logger,
		cluster:    cluster,
		elector:    cfg.Elector,
		blockers:   make([]*BlockedRecovery, 0),
		blockerTTL: cfg.ReplicaSetRecoveryBlockTime,
		stop:       make(chan struct{}, 1),
	}
}

func (f *promoteFailover) Serve(stream AnalysisReadStream) {
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

func (f *promoteFailover) Shutdown() {
	f.stop <- struct{}{}
}

func (f *promoteFailover) shouldBeAnalysisChecked() bool {
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

func (f *promoteFailover) checkAndRecover(ctx context.Context, analysis *ReplicationAnalysis) {
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

func (f *promoteFailover) promoteFollowerToMaster(ctx context.Context, analysis *ReplicationAnalysis) *Recovery {
	badSet := analysis.Set
	logger := f.logger.With().Str("ReplicaSet", string(badSet.UUID)).Logger()

	if f.hasBlockedRecovery(badSet.UUID) {
		logger.Warn().Msg("ReplicaSet has been recovered recently so new failover is blocked")
		return nil
	}

	recv := NewRecovery(analysis)
	defer func() {
		recv.EndTimestamp = util.Timestamp()
	}()

	candidateUUID, err := f.elector.ChooseMaster(badSet)
	if err != nil {
		logger.Err(err).Msg("Failed to elect a new master")
		return recv
	}
	recv.SuccessorUUID = candidateUUID

	logger.Info().Msgf("New master is elected: %s. Going to update cluster configuration", candidateUUID)

	// First priority is updating the configuration of the new master.
	// If any error, exit from the recovery.
	candidate, _ := f.cluster.Instance(candidateUUID)
	conn := f.cluster.ConnInstance(&candidate)
	applier := confApplier{
		role:          roleSuccessor,
		setUUID:       badSet.UUID,
		failedUUID:    badSet.MasterUUID,
		candidateUUID: candidateUUID,
		conn:          conn,
	}
	err = applier.apply(ctx)
	if err == nil {
		logger.Info().Msgf("Configuration of the chosen master '%s' was updated", candidateUUID)
	} else {
		logger.Err(err).Msgf("Recovery fatal error: failed to update the configuration of the chosen master '%s'", candidateUUID)
		return recv
	}

	instances := f.cluster.Instances()
	sort.Sort(NewInstanceFailoverSorter(instances))

	// Update the configuration of all the cluster members.
	for i := range instances {
		inst := &instances[i]

		if inst.UUID == candidateUUID {
			continue
		}

		conn := f.cluster.ConnInstance(inst)
		applier := confApplier{
			role:          roleReplica,
			setUUID:       badSet.UUID,
			failedUUID:    badSet.MasterUUID,
			candidateUUID: candidateUUID,
			conn:          conn,
		}
		if inst.UUID == badSet.MasterUUID {
			applier.role = roleFailed
		}

		err = applier.apply(ctx)
		if err == nil {
			logger.Info().Msgf("Configuration was updated on node '%s'", inst.UUID)
		} else {
			logger.Err(err).Msgf("Failed to update configuration on node '%s'", inst.UUID)
		}
	}

	routers := f.cluster.Routers()
	for i := range routers {
		r := &routers[i]
		conn := f.cluster.ConnRouter(r)
		applier := confApplier{
			role:          roleRouter,
			setUUID:       badSet.UUID,
			failedUUID:    badSet.MasterUUID,
			candidateUUID: candidateUUID,
			conn:          conn,
		}
		err := applier.apply(ctx)
		if err == nil {
			logger.Info().Msgf("Configuration was updated on router '%s'", r.UUID)
		} else {
			logger.Err(err).Msgf("Failed to update configuration on router '%s'", r.UUID)
		}
	}

	recv.IsSuccessful = true
	return recv
}

func (f *promoteFailover) registryRecovery(r *Recovery) {
	blocker := NewBlockedRecovery(r, f.blockerTTL)

	f.blockerSync.Lock()
	f.blockers = append(f.blockers, blocker)
	f.blockerSync.Unlock()
}

func (f *promoteFailover) hasBlockedRecovery(uuid vshard.ReplicaSetUUID) bool {
	f.blockerSync.RLock()
	defer f.blockerSync.RUnlock()

	for _, b := range f.blockers {
		if b.Recovery.SetUUID == uuid && !b.Expired() {
			return true
		}
	}

	return false
}

func (f *promoteFailover) cleanup(force bool) {
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

// confApplier applies instance role during the failover/switchover.
type confApplier struct {
	role          recoveryRole
	setUUID       vshard.ReplicaSetUUID
	failedUUID    vshard.InstanceUUID
	candidateUUID vshard.InstanceUUID
	conn          *vshard.Connector
}

func (ca confApplier) apply(ctx context.Context) error {
	call := ca.buildQuery()
	resp := ca.conn.Exec(ctx, call)
	return resp.Error
}

func (ca confApplier) isReadOnlyRole() bool {
	return ca.role != roleSuccessor
}

func (ca confApplier) buildQuery() tarantool.Query {
	if ca.role == roleRouter {
		return ca.buildRouterQuery()
	}
	return ca.buildStorageQuery()
}

func (ca confApplier) buildRouterQuery() tarantool.Query {
	return &tarantool.Call{
		Name: "qumomf_change_master",
		Tuple: []interface{}{
			string(ca.setUUID), string(ca.failedUUID), string(ca.candidateUUID),
		},
	}
}

func (ca confApplier) buildStorageQuery() tarantool.Query {
	ro := ca.isReadOnlyRole()

	call := &tarantool.Eval{
		Expression: `
			local arg = {...}

			qumomf_change_master(arg[1], arg[2], arg[3])
			box.cfg({
        		read_only = arg[4],
    		})
		`,
		Tuple: []interface{}{
			string(ca.setUUID), string(ca.failedUUID), string(ca.candidateUUID), ro,
		},
	}

	return call
}
