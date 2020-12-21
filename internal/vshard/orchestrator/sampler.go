package orchestrator

import (
	"sync"

	"github.com/rs/zerolog"
)

type sampler struct {
	enabled      bool
	fingerprints map[string]string
	mu           *sync.RWMutex
}

func (s *sampler) sample(analysis *ReplicationAnalysis) zerolog.Level {
	if !s.enabled {
		return zerolog.InfoLevel
	}

	got, err := analysis.GetHash()
	if err != nil {
		return zerolog.InfoLevel
	}
	s.mu.RLock()
	found, ok := s.fingerprints[string(analysis.Set.UUID)]
	s.mu.RUnlock()
	if ok && found == got {
		return zerolog.DebugLevel
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.fingerprints[string(analysis.Set.UUID)] = got

	return zerolog.InfoLevel
}
