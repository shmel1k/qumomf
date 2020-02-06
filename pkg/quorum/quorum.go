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
	ChooseMaster(vshard.ShardInfo) (string, error)
}

type lagQuorum struct {
}

func (*lagQuorum) ChooseMaster(info vshard.ShardInfo) (string, error) {
	if len(info) == 0 {
		return "", ErrEmptyInfo
	}

	minLag := maxLag
	minUUID := ""
	for _, v := range info {
		if v.Lag < minLag && v.Status == vshard.StatusFollow {
			minLag = v.Lag
			minUUID = v.UUID
		}
	}

	if minUUID == "" {
		return "", ErrNoReplicaFound
	}

	return minUUID, nil
}
