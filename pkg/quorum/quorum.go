package quorum

import (
	"errors"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

var (
	ErrNoFollowers      = errors.New("quorum: ReplicaSet does not have any followers")
	ErrNoCandidateFound = errors.New("quorum: no available candidate found")
)

const (
	maxLag = float64(1000)
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
	followers := set.Followers()
	if len(followers) == 0 {
		return "", ErrNoFollowers
	}

	minLag := maxLag
	minUUID := vshard.InstanceUUID("")
	for i := range followers {
		r := &followers[i]
		upstream := r.Upstream
		if upstream == nil {
			continue
		}
		if upstream.Lag < minLag {
			minLag = upstream.Lag
			minUUID = r.UUID
		}
	}

	if minUUID == "" {
		return "", ErrNoCandidateFound
	}

	return minUUID, nil
}
