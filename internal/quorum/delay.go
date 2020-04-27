package quorum

import (
	"math"

	"github.com/shmel1k/qumomf/internal/vshard"
)

const (
	maxLag = math.MaxFloat64
)

type delayElector struct {
}

// NewDelayElector returns a new elector based on replica's idle value.
//
// This elector chooses the candidate to be a master selecting
// the replica with a minimum idle/lag value.
func NewDelayElector() Elector {
	return &delayElector{}
}

func (*delayElector) ChooseMaster(set vshard.ReplicaSet) (vshard.InstanceUUID, error) {
	followers := set.AliveFollowers()
	if len(followers) == 0 {
		return "", ErrNoAliveFollowers
	}

	minLag := maxLag
	minUUID := vshard.InstanceUUID("")
	for i := range followers {
		r := &followers[i]
		repl := &r.StorageInfo.Replication

		if repl.Delay < minLag {
			minLag = repl.Delay
			minUUID = r.UUID
		}
	}

	if minUUID == "" {
		return "", ErrNoCandidateFound
	}

	return minUUID, nil
}

func (*delayElector) Mode() Mode {
	return ModeDelay
}
