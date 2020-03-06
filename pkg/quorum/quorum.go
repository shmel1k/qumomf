package quorum

import (
	"errors"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

var (
	ErrEmptyInfo      = errors.New("quorum: empty shard info given")
	ErrNoReplicaFound = errors.New("quorum: no available replica found")
)

const (
	maxLag = float64(100)
)

type Quorum interface {
	// ChooseMaster selects new master and returns back its uuid
	ChooseMaster(vshard.ReplicaSetInfo) (vshard.ReplicaUUID, error)
}

type lagQuorum struct {
}

func NewLagQuorum() Quorum {
	return &lagQuorum{}
}

func (*lagQuorum) ChooseMaster(info vshard.ReplicaSetInfo) (vshard.ReplicaUUID, error) {
	if len(info) == 0 {
		return "", ErrEmptyInfo
	}

	minLag := maxLag
	minUUID := vshard.ReplicaUUID("")
	for _, r := range info {
		if r.Lag < minLag && r.Status == vshard.StatusFollow {
			minLag = r.Lag
			minUUID = r.UUID
		}
	}

	if minUUID == "" {
		return "", ErrNoReplicaFound
	}

	return minUUID, nil
}
