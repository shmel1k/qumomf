package quorum

import (
	"errors"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

type Mode string

const (
	ModeDelay Mode = "delay"
	ModeSmart Mode = "smart"
)

var (
	ErrNoAliveFollowers = errors.New("quorum: ReplicaSet does not have any alive followers")
	ErrNoCandidateFound = errors.New("quorum: no available candidate found")
)

type Elector interface {
	// ChooseMaster selects new master and returns back its uuid.
	ChooseMaster(set vshard.ReplicaSet) (vshard.InstanceUUID, error)
	// Mode returns the elector type.
	Mode() Mode
}

func New(m Mode) Elector {
	switch m {
	case ModeDelay:
		return NewDelayElector()
	case ModeSmart:
		return NewSmartElector()
	}

	panic("This code should not be reached!")
}
