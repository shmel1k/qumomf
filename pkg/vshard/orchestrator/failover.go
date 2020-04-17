package orchestrator

import (
	"context"
	"sort"
	"strings"
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

const (
	// recoveryLua is a template of Lua script which should be executed
	// on every cluster nodes during the failover.
	recoveryLua = `
		log = require('log')

		log.warn("qumomf: start recovery")

		local cfg = {}
		local is_storage = vshard.router.internal.static_router == nil 
		if is_storage then
			cfg = table.deepcopy(vshard.storage.internal.current_cfg)
		else
			cfg = table.deepcopy(vshard.router.internal.static_router.current_cfg)
		end

		for replica_uuid in pairs(cfg.sharding["{set_uuid}"].replicas) do
			is_master = replica_uuid == "{new_master_uuid}"
			cfg.sharding["{set_uuid}"].replicas[replica_uuid].master = is_master 
		end

		if is_storage then
			log.warn("qumomf: apply new vshard configuration on storage")
			local this_uuid = vshard.storage.internal.this_replica.uuid
			vshard.storage.cfg(cfg, this_uuid)

			if vshard.storage.internal.this_replicaset.uuid == "{set_uuid}" then 
				box.cfg({
					read_only = this_uuid ~= "{new_master_uuid}",
				})
			end
		else
			log.warn("qumomf: apply new vshard configuration on router")
			vshard.router.cfg(cfg)
		end
		log.warn("qumomf: end recovery")
	`
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

func NewPromoteFailover(cluster *vshard.Cluster, cfg FailoverConfig, logger zerolog.Logger) Failover {
	return &promoteFailover{
		cluster:    cluster,
		elector:    cfg.Elector,
		blockers:   make([]*BlockedRecovery, 0),
		blockerTTL: cfg.ReplicaSetRecoveryBlockTime,
		stop:       make(chan struct{}, 1),
		logger:     logger,
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

	recvQuery := buildRecoveryQuery(badSet.UUID, candidateUUID)

	// First priority is updating the configuration of the new master.
	// If any error, exit from the recovery.
	candidate, _ := f.cluster.Instance(candidateUUID)
	conn := f.cluster.Connector(candidate.URI)
	resp := conn.Exec(ctx, recvQuery)
	if resp.Error == nil {
		logger.Info().Msgf("Configuration of the chosen master '%s' was updated", candidateUUID)
	} else {
		logger.Err(resp.Error).Msgf("Recovery fatal error: failed to update the configuration of the chosen master '%s'", candidateUUID)
		return recv
	}

	// Update routers configuration to accept write requests as quickly as possible.
	routers := f.cluster.Routers()
	for i := range routers {
		r := &routers[i]
		conn := f.cluster.Connector(r.URI)
		resp := conn.Exec(ctx, recvQuery)
		if resp.Error == nil {
			logger.Info().Msgf("Configuration was updated on router '%s'", r.UUID)
		} else {
			logger.Err(resp.Error).Msgf("Failed to update configuration on router '%s'", r.UUID)
		}
	}

	instances := f.cluster.Instances()
	sort.Sort(NewInstanceFailoverSorter(instances))

	// Update the configuration of all the cluster members.
	for i := range instances {
		inst := &instances[i]

		if inst.UUID == candidateUUID {
			continue
		}

		conn := f.cluster.Connector(inst.URI)
		resp := conn.Exec(ctx, recvQuery)
		if resp.Error == nil {
			logger.Info().Msgf("Configuration was updated on node '%s'", inst.UUID)
		} else {
			logger.Err(resp.Error).Msgf("Failed to update configuration on node '%s'", inst.UUID)
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

func buildRecoveryQuery(set vshard.ReplicaSetUUID, candidate vshard.InstanceUUID) tarantool.Query {
	lua := generateRecoveryLua(set, candidate)
	return &tarantool.Eval{
		Expression: lua,
	}
}

func generateRecoveryLua(set vshard.ReplicaSetUUID, candidate vshard.InstanceUUID) string {
	lua := strings.ReplaceAll(recoveryLua, "{set_uuid}", string(set))
	lua = strings.ReplaceAll(lua, "{new_master_uuid}", string(candidate))

	return lua
}
