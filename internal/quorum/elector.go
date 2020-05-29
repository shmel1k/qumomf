package quorum

import (
	"errors"
	"fmt"

	"github.com/shmel1k/qumomf/internal/vshard"
)

type Mode string

const (
	ModeIdle  Mode = "idle"
	ModeSmart Mode = "smart"
)

var (
	ErrNoAliveFollowers = errors.New("quorum: replica set does not have any alive followers or all of them were excluded from the election")
	ErrNoCandidateFound = errors.New("quorum: no available candidate found")
)

type Options struct {
	ReasonableFollowerLSNLag int64
	ReasonableFollowerIdle   float64
}

type Elector interface {
	// ChooseMaster selects new master and returns back its uuid.
	ChooseMaster(set vshard.ReplicaSet) (vshard.InstanceUUID, error)
	// Mode returns the elector type.
	Mode() Mode
}

func New(m Mode, opts Options) Elector {
	switch m {
	case ModeIdle:
		return NewIdleElector(opts)
	case ModeSmart:
		return NewSmartElector(opts)
	}

	panic(fmt.Sprintf("Elector: got unknown mode %s", m))
}

// filter filters out the instances which must not be promoted to the master.
func filter(instances []vshard.Instance, opts Options) []vshard.Instance {
	filtered := make([]vshard.Instance, 0, len(instances))

	for i := range instances {
		inst := &instances[i]

		// Exclude all followers with negative priority.
		if inst.Priority < 0 {
			continue
		}

		if opts.ReasonableFollowerLSNLag != 0 {
			// Exclude followers too far from the master.
			if inst.LSNBehindMaster > opts.ReasonableFollowerLSNLag {
				continue
			}
		}

		if opts.ReasonableFollowerIdle != 0 {
			// Exclude followers too far from the master.
			if inst.Idle() > opts.ReasonableFollowerIdle {
				continue
			}
		}

		filtered = append(filtered, *inst)
	}

	return filtered
}
