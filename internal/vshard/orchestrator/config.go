package orchestrator

import (
	"time"

	"github.com/shmel1k/qumomf/internal/quorum"
)

type Config struct {
	RecoveryPollTime  time.Duration
	DiscoveryPollTime time.Duration
}

type FailoverConfig struct {
	Hooker                      *Hooker
	Elector                     quorum.Elector
	InstanceRecoveryBlockTime   time.Duration
	ReplicaSetRecoveryBlockTime time.Duration
}
