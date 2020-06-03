package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/viciious/go-tarantool"

	"github.com/shmel1k/qumomf/internal/quorum"
	"github.com/shmel1k/qumomf/internal/util"
	"github.com/shmel1k/qumomf/internal/vshard"
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
			log.warn("qumomf: apply new vshard configuration to storage")
			local this_uuid = box.info.uuid
			vshard.storage.cfg(cfg, this_uuid)

			if box.info.cluster.uuid == "{set_uuid}" then 
				box.cfg({
					read_only = this_uuid ~= "{new_master_uuid}",
				})
			end
		else
			log.warn("qumomf: apply new vshard configuration to router")
			vshard.router.cfg(cfg)
		end
		log.warn("qumomf: end recovery")
	`
)

type Failover interface {
	Serve(stream AnalysisReadStream)
	Shutdown()
}

type failover struct {
	cluster *vshard.Cluster
	elector quorum.Elector
	hooker  *Hooker

	recoveries      []*Recovery
	recvSync        sync.RWMutex
	recvSetTTL      time.Duration
	recvInstanceTTL time.Duration

	stop   chan struct{}
	logger zerolog.Logger
}

func NewDefaultFailover(cluster *vshard.Cluster, cfg FailoverConfig, logger zerolog.Logger) Failover {
	return &failover{
		cluster:         cluster,
		elector:         cfg.Elector,
		hooker:          cfg.Hooker,
		recoveries:      make([]*Recovery, 0),
		recvSetTTL:      cfg.ReplicaSetRecoveryBlockTime,
		recvInstanceTTL: cfg.InstanceRecoveryBlockTime,
		stop:            make(chan struct{}, 1),
		logger:          logger,
	}
}

func (f *failover) Serve(stream AnalysisReadStream) {
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

func (f *failover) Shutdown() {
	f.stop <- struct{}{}
}

func (f *failover) shouldBeAnalysisChecked() bool {
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

func (f *failover) checkAndRecover(ctx context.Context, analysis *ReplicationAnalysis) {
	logger := f.logger.With().Str("ReplicaSet", string(analysis.Set.UUID)).Logger()
	logger.Info().Msgf("checkAndRecover: %s", analysis.String())

	recvFunc, desc := f.getCheckAndRecoveryFunc(analysis.State)
	if recvFunc == nil {
		if desc != "" {
			logger.Warn().Msg(desc)
		}
		return
	}

	f.cluster.StartRecovery()
	logger.Warn().Msg(desc)
	logger.Info().Msgf("Cluster snapshot before recovery: %s", f.cluster.Dump())
	recoveries := recvFunc(ctx, analysis)
	for _, recv := range recoveries {
		f.registryRecovery(recv)

		if recv.IsSuccessful {
			_ = f.hooker.ExecuteProcesses(HookPostSuccessfulFailover, recv, false)
		} else {
			_ = f.hooker.ExecuteProcesses(HookPostUnsuccessfulFailover, recv, false)
		}

		logger.Info().Msgf("Finished recovery: %s", recv)
	}
	if len(recoveries) > 0 {
		logger.Info().Msg("Run a force discovery after applied recoveries")
		f.cluster.Discover()
		logger.Info().Msgf("Cluster snapshot after recovery: %s", f.cluster.Dump())
	}
	f.cluster.StopRecovery()
}

func (f *failover) getCheckAndRecoveryFunc(state ReplicaSetState) (rf RecoveryFunc, desc string) {
	switch state {
	case NoProblem:
		// Nothing to do, everything is OK.
	case DeadMaster:
		rf = f.promoteFollowerToMaster
		desc = "Master cannot be reached by qumomf. Will run failover."
	case DeadMasterAndSomeFollowers:
		rf = f.promoteFollowerToMaster
		desc = "Master cannot be reached by qumomf and some of its followers are unreachable. Will run failover."
	case DeadMasterAndFollowers:
		desc = "Master cannot be reached by qumomf and none of its followers is replicating. No actions will be applied."
	case AllMasterFollowersNotReplicating:
		desc = "Master is reachable but none of its replicas is replicating. No actions will be applied."
	case DeadMasterWithoutFollowers:
		desc = "Master cannot be reached by qumomf and has no followers. No actions will be applied."
	case DeadFollowers:
		desc = "Master is reachable but some of its replicas are not replicating. No actions will be applied."
	case NetworkProblems:
		desc = "Master cannot be reached by qumomf but some followers are still replicating. It might be a network problem, no actions will be applied."
	case MasterMasterReplication:
		rf = f.applyFollowerRoleToCoMasters
		desc = "Found master-master topology. Will apply follower role to all co-masters except a shard leader."
	case InconsistentVShardConfiguration:
		desc = "Found replicas with inconsistent vshard topology. No actions will be applied."
	default:
		panic(fmt.Sprintf("Unknown analysis state: %s", state))
	}

	return
}

func (f *failover) promoteFollowerToMaster(ctx context.Context, analysis *ReplicationAnalysis) []*Recovery {
	badSet := analysis.Set
	logger := f.logger.With().Str("ReplicaSet", string(badSet.UUID)).Logger()

	if f.hasBlockedRecovery(string(badSet.UUID)) {
		logger.Warn().Msg("ReplicaSet has been recovered recently so new failover is blocked")
		return nil
	}

	failed, _ := badSet.Master()
	recv := NewRecovery(RecoveryScopeSet, failed.Ident(), *analysis)
	recv.ExpireAfter(f.recvSetTTL)
	recv.ClusterName = f.cluster.Name
	defer func() {
		recv.EndTimestamp = util.Timestamp()
	}()

	err := f.hooker.ExecuteProcesses(HookPreFailover, recv, true)
	if err != nil {
		return []*Recovery{recv}
	}

	candidateUUID, err := f.elector.ChooseMaster(badSet)
	if err != nil {
		logger.Err(err).Msg("Failed to elect a new master")
		return []*Recovery{recv}
	}

	candidate, _ := f.cluster.Instance(candidateUUID)
	recv.Successor = candidate.Ident()
	if ok, reason := f.shouldPromoteFollower(candidate); !ok {
		logger.Warn().Msgf("Promotion of the chosen candidate is too complex. The recovery is interrupted. Reason: %s", reason)
		return []*Recovery{recv}
	}

	logger.Info().Msgf("New master is elected: %s. Going to update cluster configuration", candidateUUID)

	recvQuery := buildRecoveryQuery(badSet.UUID, candidateUUID)

	// First priority is updating the configuration of the new master.
	// If any error, exit from the recovery.
	conn := f.cluster.Connector(candidate.URI)
	resp := conn.Exec(ctx, recvQuery)
	if resp.Error == nil {
		logger.Info().
			Str("URI", candidate.URI).
			Str("UUID", string(candidateUUID)).
			Msg("Configuration of the chosen master was updated")
	} else {
		logger.Err(resp.Error).
			Str("URI", candidate.URI).
			Str("UUID", string(candidateUUID)).
			Msg("Recovery fatal error: failed to update the configuration of the chosen master")

		return []*Recovery{recv}
	}

	// Update routers configuration to accept write requests as quickly as possible.
	routers := f.cluster.Routers()
	for i := range routers {
		r := &routers[i]
		conn := f.cluster.Connector(r.URI)
		resp := conn.Exec(ctx, recvQuery)
		if resp.Error == nil {
			logger.Info().
				Str("URI", r.URI).
				Str("UUID", string(r.UUID)).
				Msg("Configuration was updated on router")
		} else {
			logger.Err(resp.Error).
				Str("URI", r.URI).
				Str("UUID", string(r.UUID)).
				Msg("Failed to update configuration on router")
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
			logger.Info().
				Str("URI", inst.URI).
				Str("UUID", string(inst.UUID)).
				Msg("Configuration was updated on node")
		} else {
			logger.Err(resp.Error).
				Str("URI", inst.URI).
				Str("UUID", string(inst.UUID)).
				Msg("Failed to update configuration on node")
		}
	}

	recv.IsSuccessful = true
	return []*Recovery{recv}
}

// shouldPromoteFollower performs some checks of the chosen candidate to ensure
// that failover will not make the shard state even worse.
//
// Sometimes it is better to give up and allow leather bags to do their job.
func (f *failover) shouldPromoteFollower(inst vshard.Instance) (ok bool, reason string) {
	if inst.LSNBehindMaster < 0 {
		return false, "master LSN is behind the candidate LSN: replication might was broken before the crash"
	}

	upstreamStatus := inst.Upstream.Status
	if upstreamStatus != vshard.UpstreamFollow && upstreamStatus != vshard.UpstreamRunning {
		return false, "candidate had neither an upstream status Follow nor Running before the crash"
	}

	return true, ""
}

// applyFollowerRoleToCoMasters applies follower role to all masters in the shard except the leader.
func (f *failover) applyFollowerRoleToCoMasters(ctx context.Context, analysis *ReplicationAnalysis) []*Recovery {
	badSet := &analysis.Set
	logger := f.logger.With().Str("ReplicaSet", string(badSet.UUID)).Logger()

	recvQuery := buildRecoveryQuery(badSet.UUID, badSet.MasterUUID)

	master, _ := badSet.Master()
	followers := badSet.Followers()
	recoveries := make([]*Recovery, 0)
	for i := range followers {
		inst := &followers[i]

		if inst.VShardFingerprint == master.VShardFingerprint {
			continue
		}

		if f.hasBlockedRecovery(string(inst.UUID)) {
			logger.Warn().
				Str("URI", inst.URI).
				Str("UUID", string(inst.UUID)).
				Msg("Instance has been recovered recently so new failover is blocked")

			continue
		}

		recv := NewRecovery(RecoveryScopeInstance, inst.Ident(), *analysis)
		recv.ExpireAfter(f.recvInstanceTTL)
		recv.ClusterName = f.cluster.Name

		err := f.hooker.ExecuteProcesses(HookPreFailover, recv, true)
		if err != nil {
			recv.EndTimestamp = util.Timestamp()
			recoveries = append(recoveries, recv)

			continue
		}

		conn := f.cluster.Connector(inst.URI)
		resp := conn.Exec(ctx, recvQuery)
		if resp.Error == nil {
			logger.Info().
				Str("URI", inst.URI).
				Str("UUID", string(inst.UUID)).
				Msg("Configuration was updated on node")
			recv.IsSuccessful = true
		} else {
			logger.Err(resp.Error).
				Str("URI", inst.URI).
				Str("UUID", string(inst.UUID)).
				Msg("Failed to update configuration on node")
		}

		recv.EndTimestamp = util.Timestamp()
		recoveries = append(recoveries, recv)
	}

	return recoveries
}

func (f *failover) registryRecovery(r *Recovery) {
	f.recvSync.Lock()
	f.recoveries = append(f.recoveries, r)
	f.recvSync.Unlock()
}

func (f *failover) hasBlockedRecovery(key string) bool {
	f.recvSync.RLock()
	defer f.recvSync.RUnlock()

	for _, b := range f.recoveries {
		if b.ScopeKey() == key && !b.Expired() {
			return true
		}
	}

	return false
}

func (f *failover) cleanup(force bool) {
	// It is not a frequent operation, so do not
	// see any reason to optimize this place.

	f.recvSync.RLock()
	if len(f.recoveries) == 0 {
		f.recvSync.RUnlock()
		return
	}

	alive := make([]*Recovery, 0)
	if !force {
		for _, b := range f.recoveries {
			if !b.Expired() {
				alive = append(alive, b)
			}
		}
	}
	f.recvSync.RUnlock()

	f.recvSync.Lock()
	f.recoveries = alive
	f.recvSync.Unlock()
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
