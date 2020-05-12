package quorum

import (
	"sort"

	"github.com/shmel1k/qumomf/internal/vshard"
)

type smartElector struct {
}

// NewSmartElector returns a new elector based on rules:
//  - compare vshard configuration consistency,
//  - compare upstream status,
//  - compare LSN behind the master,
//  - compare when replica got last heartbeat signal or data from master.
func NewSmartElector() Elector {
	return &smartElector{}
}

func (e *smartElector) ChooseMaster(set vshard.ReplicaSet) (vshard.InstanceUUID, error) {
	followers := set.AliveFollowers()
	if len(followers) == 0 {
		return "", ErrNoAliveFollowers
	}

	master, err := set.Master()
	if err != nil {
		return "", err
	}
	sorter := newInstanceSorter(master, followers)
	sort.Sort(sorter)

	return followers[0].UUID, nil
}

func (e *smartElector) Mode() Mode {
	return ModeSmart
}

// instanceSorter sorts instances by their priority to be a new master.
type instanceSorter struct {
	master    vshard.Instance
	instances []vshard.Instance
}

func newInstanceSorter(master vshard.Instance, instances []vshard.Instance) *instanceSorter {
	return &instanceSorter{
		master:    master,
		instances: instances,
	}
}

func (s *instanceSorter) Len() int {
	return len(s.instances)
}

func (s *instanceSorter) Swap(i, j int) {
	s.instances[i], s.instances[j] = s.instances[j], s.instances[i]
}

func (s *instanceSorter) Less(i, j int) bool {
	left, right := s.instances[i], s.instances[j]

	// Prefer replicas with the same vshard configuration as master.
	confHash := s.master.VShardFingerprint
	if left.VShardFingerprint == confHash && right.VShardFingerprint != confHash {
		return true
	}
	if left.VShardFingerprint != confHash && right.VShardFingerprint == confHash {
		return false
	}

	// Prefer replicas which have follow upstream status.
	if left.Upstream.Status == vshard.UpstreamFollow && right.Upstream.Status != vshard.UpstreamFollow {
		return true
	}
	if left.Upstream.Status != vshard.UpstreamFollow && right.Upstream.Status == vshard.UpstreamFollow {
		return false
	}

	// Prefer most up to date replica.
	if left.LSNBehindMaster != right.LSNBehindMaster {
		// Special case: when replication is broken and replica has been recovered from an old snapshot with
		// LSN in front of master LSN.
		if left.LSNBehindMaster > 0 && right.LSNBehindMaster < 0 {
			return true
		}
		if left.LSNBehindMaster < 0 && right.LSNBehindMaster > 0 {
			return false
		}

		return left.LSNBehindMaster < right.LSNBehindMaster
	}

	return left.StorageInfo.Replication.Delay < right.StorageInfo.Replication.Delay
}
