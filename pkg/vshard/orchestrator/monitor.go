package orchestrator

import (
	"time"

	"github.com/rs/zerolog"

	"github.com/shmel1k/qumomf/pkg/vshard"
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
	config  Config
	cluster *vshard.Cluster

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
				for _, set := range m.cluster.ReplicaSets() {
					go func(set vshard.ReplicaSet) {
						analysis := analyze(set, m.logger)
						if analysis != nil {
							stream <- analysis
						}
					}(set)
				}
			} else {
				m.logger.Info().Msgf("Waiting for %+v seconds to pass before running failure detection/recovery", checkAndRecoverWaitPeriod.Seconds())
			}
		}
	}
}

func analyze(set vshard.ReplicaSet, logger zerolog.Logger) *ReplicationAnalysis { //nolint: gocyclo
	master, err := set.Master()
	if err != nil {
		// Something really weird but we have data inconsistency here.
		// Master UUID not found in ReplicaSet.
		logger.Error().Msgf("Failed to analyze replicaset state: master UUID '%s' not found", set.MasterUUID)
		return nil
	}

	countReplicas := 0
	countWorkingReplicas := 0
	countReplicatingReplicas := 0
	countInconsistentVShardConf := 0
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
				logger.Warn().
					Str("ReplicaSet", string(set.UUID)).
					Msgf("Found M-M replication ('%s'-'%s') in ReplicaSet", set.MasterUUID, r.UUID)
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
		state = InconsistentVShardConfiguration
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
