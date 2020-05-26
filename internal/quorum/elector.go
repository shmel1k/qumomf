package quorum

import (
	"errors"
	"fmt"

	"github.com/shmel1k/qumomf/internal/vshard"
)

type Mode string

const (
	ModeDelay Mode = "delay"
	ModeSmart Mode = "smart"
)

var (
	ErrNoAliveFollowers = errors.New("quorum: replica set does not have any alive followers or all of them were excluded from the election")
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

	panic(fmt.Sprintf("Elector: got unknown mode %s", m))
}
