package quorum

import (
	"errors"
	"math"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

var (
	ErrNoAliveFollowers = errors.New("quorum: ReplicaSet does not have any alive followers")
	ErrNoCandidateFound = errors.New("quorum: no available candidate found")
)

const (
	maxLag = math.MaxFloat64
)

type Quorum interface {
	// ChooseMaster selects new master and returns back its uuid
	ChooseMaster(set vshard.ReplicaSet) (vshard.InstanceUUID, error)
}

type lagQuorum struct {
}

func NewLagQuorum() Quorum {
	return &lagQuorum{}
}

func (*lagQuorum) ChooseMaster(set vshard.ReplicaSet) (vshard.InstanceUUID, error) {
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
