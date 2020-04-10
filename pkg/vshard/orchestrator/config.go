package orchestrator

import (
	"time"

	"github.com/rs/zerolog"

	"github.com/shmel1k/qumomf/pkg/quorum"
)

type Config struct {
	Logger            zerolog.Logger
	RecoveryPollTime  time.Duration
	DiscoveryPollTime time.Duration
}

type FailoverConfig struct {
	Logger                      zerolog.Logger
	Elector                     quorum.Quorum
	ReplicaSetRecoveryBlockTime time.Duration
}
