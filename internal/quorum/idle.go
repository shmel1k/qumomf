package quorum

import (
	"math"

	"github.com/shmel1k/qumomf/internal/vshard"
)

const (
	maxIdle = math.MaxFloat64
)

type idleElector struct {
	opts Options
}

// NewIdleElector returns a new elector based on the follower's idle value.
//
// This elector chooses the candidate to be a master selecting
// the follower with a minimum idle value.
func NewIdleElector(opts Options) Elector {
	return &idleElector{
		opts: opts,
	}
}

func (e *idleElector) ChooseMaster(set vshard.ReplicaSet) (vshard.InstanceUUID, error) {
	followers := filter(set.AliveFollowers(), e.opts)
	if len(followers) == 0 {
		return "", ErrNoAliveFollowers
	}

	minIdle := maxIdle
	minUUID := vshard.InstanceUUID("")
	for i := range followers {
		r := &followers[i]

		if r.Idle() < minIdle {
			minIdle = r.Idle()
			minUUID = r.UUID
		}
	}

	if minUUID == "" {
		return "", ErrNoCandidateFound
	}

	return minUUID, nil
}

func (*idleElector) Mode() Mode {
	return ModeIdle
}
