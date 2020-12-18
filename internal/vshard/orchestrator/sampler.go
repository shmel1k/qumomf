package orchestrator

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/shmel1k/qumomf/internal/util"
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

	s.mu.RLock()
	got, err := util.GetHash([]byte(analysis.String()))
	s.mu.RUnlock()
	if err != nil {
		return zerolog.InfoLevel
	}

	found, ok := s.fingerprints[string(analysis.Set.UUID)]
	if ok && found == got {
		return zerolog.DebugLevel
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.fingerprints[string(analysis.Set.UUID)] = got

	return zerolog.InfoLevel
}
