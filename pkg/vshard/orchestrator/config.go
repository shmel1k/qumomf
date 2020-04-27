package orchestrator

import (
	"time"

	"github.com/shmel1k/qumomf/pkg/quorum"
)

type Config struct {
	RecoveryPollTime  time.Duration
	DiscoveryPollTime time.Duration
}

type FailoverConfig struct {
	Elector                     quorum.Elector
	InstanceRecoveryBlockTime   time.Duration
	ReplicaSetRecoveryBlockTime time.Duration
}
