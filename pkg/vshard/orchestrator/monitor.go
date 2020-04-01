package orchestrator

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

type Monitor interface {
	Serve() AnalysisReadStream
	Shutdown()
}

func NewMonitor(cfg Config, cluster *vshard.Cluster) Monitor {
	return &storageMonitor{
		config:  cfg,
		cluster: cluster,
		stop:    make(chan struct{}, 1),
	}
}

type storageMonitor struct {
	config  Config
	cluster *vshard.Cluster
	stop    chan struct{}
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
						analysis := m.analyze(set)
						if analysis != nil {
							stream <- analysis
						}
					}(set)
				}
			} else {
				log.Debug().Msgf("Waiting for %+v seconds to pass before running failure detection/recovery", checkAndRecoverWaitPeriod.Seconds())
			}
		}
	}
}

func (m *storageMonitor) analyze(set vshard.ReplicaSet) *ReplicationAnalysis {
	// TODO: make it smarter - https://github.com/shmel1k/qumomf/issues/3

	countReplicas := 0
	countWorkingReplicas := 0
	countReplicatingReplicas := 0
	followers := set.Followers()
	for i := range followers {
		r := &followers[i]
		countReplicas++
		if r.LastCheckValid {
			countWorkingReplicas++

			if !r.HasAlert(vshard.AlertUnreachableMaster) {
				countReplicatingReplicas++
			}
		}
	}

	master, err := set.Master()
	if err != nil {
		// Something really weird but we have data inconsistency here.
		// Master UUID not found in ReplicaSet.
		log.Error().Msgf("Failed to analyze replicaset state: master UUID '%s' not found", set.MasterUUID)
		return nil
	}
	isMasterDead := !master.LastCheckValid

	state := NoProblem
	if isMasterDead && countWorkingReplicas == countReplicas && countReplicatingReplicas == countReplicas {
		if countReplicas == 0 {
			state = DeadMasterWithoutFollowers
		} else {
			state = DeadMaster
		}
	} else if isMasterDead && countWorkingReplicas <= countReplicas && countReplicatingReplicas < countReplicas {
		if countWorkingReplicas == 0 {
			state = DeadMasterAndFollowers
		} else {
			state = DeadMasterAndSomeFollowers
		}
	} else if countReplicas > 0 && countReplicatingReplicas == 0 {
		state = AllMasterFollowersNotReplicating
	}

	return &ReplicationAnalysis{
		Set:                      set,
		CountReplicas:            countReplicas,
		CountWorkingReplicas:     countWorkingReplicas,
		CountReplicatingReplicas: countReplicatingReplicas,
		State:                    state,
	}
}

func (m *storageMonitor) Shutdown() {
	m.stop <- struct{}{}
}
