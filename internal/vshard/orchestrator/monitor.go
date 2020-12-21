package orchestrator

import (
	"time"

	"github.com/rs/zerolog"

	"github.com/shmel1k/qumomf/internal/metrics"
	"github.com/shmel1k/qumomf/internal/vshard"
)

type Monitor interface {
	Serve() AnalysisReadStream
	Shutdown()
}

func NewMonitor(cluster *vshard.Cluster, cfg Config, logger zerolog.Logger) Monitor {
	return &storageMonitor{
		config:  cfg,
		cluster: cluster,
		stop:    make(chan struct{}, 1),
		logger:  logger,
	}
}

type storageMonitor struct {
	config Config

	cluster  *vshard.Cluster
	analyzed int64 // identifier of the last analyzed cluster topology

	stop   chan struct{}
	logger zerolog.Logger
}

func (m *storageMonitor) Serve() AnalysisReadStream {
	stream := NewAnalysisStream()
	go m.continuousDiscovery(stream)

	return stream
}

func (m *storageMonitor) continuousDiscovery(stream AnalysisWriteStream) {
	recoveryTick := time.NewTicker(m.config.RecoveryPollTime)
	defer recoveryTick.Stop()
	discoveryTick := time.NewTicker(m.config.DiscoveryPollTime)
	defer discoveryTick.Stop()

	continuousDiscoveryStartTime := time.Now()
	checkAndRecoverWaitPeriod := 3 * m.config.DiscoveryPollTime

	runCheckAndRecoverOperationsTimeRipe := func() bool {
		return time.Since(continuousDiscoveryStartTime) >= checkAndRecoverWaitPeriod
	}

	for {
		select {
		case <-m.stop:
			return
		case <-discoveryTick.C:
			go m.cluster.Discover()
		case <-recoveryTick.C:
			// NOTE: we might improve this place checking the delay only on start.
			if runCheckAndRecoverOperationsTimeRipe() {
				m.checkCluster(stream)
			} else {
				m.logger.Info().Msgf("Waiting for %+v seconds to pass before running failure detection/recovery", checkAndRecoverWaitPeriod.Seconds())
			}
		}
	}
}

func (m *storageMonitor) checkCluster(stream AnalysisWriteStream) {
	discovered := m.cluster.LastDiscovered()
	if discovered <= m.analyzed {
		// Prevent too much analyzes of the same cluster topology.
		return
	}

	for _, set := range m.cluster.ReplicaSets() {
		go func(set vshard.ReplicaSet) {
			logger := m.logger.With().Str("replica_set", string(set.UUID)).Logger()
			analysis := analyze(set, logger)
			if analysis != nil {
				stream <- analysis

				for _, state := range ReplicaSetStateEnum {
					active := state == analysis.State
					metrics.SetShardState(m.cluster.Name, string(set.UUID), string(state), active)
				}
			}
		}(set)
	}

	m.analyzed = discovered
}

func analyze(set vshard.ReplicaSet, logger zerolog.Logger) *ReplicationAnalysis { //nolint: gocyclo
	master, err := set.Master()
	if err != nil {
		// Something really weird but we have data inconsistency here.
		// Master UUID not found in ReplicaSet.
		logger.Error().Msgf("Fatal analyze error: master '%s' not found in given snapshot. Likely an internal error", set.MasterUUID)
		return nil
	}

	countReplicas := 0
	countWorkingReplicas := 0
	countReplicatingReplicas := 0
	countInconsistentVShardConf := 0
	masterMasterReplication := false
	followers := set.Followers()
	for i := range followers {
		r := &followers[i]
		countReplicas++
		if r.LastCheckValid {
			countWorkingReplicas++

			status := r.StorageInfo.Replication.Status
			if status == vshard.StatusFollow {
				countReplicatingReplicas++
			} else if status == vshard.StatusMaster {
				countReplicatingReplicas++
				masterMasterReplication = true

				logger.Warn().Msgf("Found M-M replication ('%s'-'%s'), ('%s'-'%s')", set.MasterUUID, r.UUID, set.MasterURI, r.URI)
			}

			if r.VShardFingerprint != master.VShardFingerprint {
				countInconsistentVShardConf++
			}
		}
	}

	isMasterDead := !master.LastCheckValid // relative to qumomf

	state := NoProblem
	if isMasterDead && countWorkingReplicas == countReplicas && countReplicatingReplicas == 0 {
		if countReplicas == 0 {
			state = DeadMasterWithoutFollowers
		} else {
			state = DeadMaster
		}
	} else if isMasterDead && countWorkingReplicas <= countReplicas && countReplicatingReplicas == 0 {
		if countWorkingReplicas == 0 {
			state = DeadMasterAndFollowers
		} else {
			state = DeadMasterAndSomeFollowers
		}
	} else if isMasterDead && countReplicatingReplicas != 0 {
		state = NetworkProblems
	} else if !isMasterDead && countReplicas > 0 && countReplicatingReplicas == 0 {
		state = AllMasterFollowersNotReplicating
	} else if countInconsistentVShardConf > 0 {
		if masterMasterReplication {
			state = MasterMasterReplication
		} else {
			state = InconsistentVShardConfiguration
		}
	} else if !isMasterDead && countReplicas > 0 && countReplicatingReplicas < countReplicas {
		state = DeadFollowers
	}

	return &ReplicationAnalysis{
		Set:                         set,
		CountReplicas:               countReplicas,
		CountWorkingReplicas:        countWorkingReplicas,
		CountReplicatingReplicas:    countReplicatingReplicas,
		CountInconsistentVShardConf: countInconsistentVShardConf,
		State:                       state,
	}
}

func (m *storageMonitor) Shutdown() {
	m.stop <- struct{}{}
}
