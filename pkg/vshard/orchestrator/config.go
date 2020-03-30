package orchestrator

import "time"

type Config struct {
	RecoveryPollTime  time.Duration
	DiscoveryPollTime time.Duration
}
