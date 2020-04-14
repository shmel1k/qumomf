package orchestrator

import "github.com/shmel1k/qumomf/pkg/vshard"

// InstanceFailoverSorter sorts instances by priority to update vshard configuration.
type InstanceFailoverSorter struct {
	instances []vshard.Instance
}

func NewInstanceFailoverSorter(instances []vshard.Instance) *InstanceFailoverSorter {
	return &InstanceFailoverSorter{
		instances: instances,
	}
}

func (s *InstanceFailoverSorter) Len() int {
	return len(s.instances)
}

func (s *InstanceFailoverSorter) Swap(i, j int) {
	s.instances[i], s.instances[j] = s.instances[j], s.instances[i]
}

func (s *InstanceFailoverSorter) Less(i, j int) bool {
	left, right := s.instances[i], s.instances[j]

	// Prefer replicas which was polled successfully last time.
	if left.LastCheckValid && !right.LastCheckValid {
		return true
	}
	// Prefer instance which has unreachable master.
	if left.HasAlert(vshard.AlertUnreachableMaster) && !right.HasAlert(vshard.AlertUnreachableMaster) {
		return true
	}
	if right.HasAlert(vshard.AlertUnreachableMaster) && !left.HasAlert(vshard.AlertUnreachableMaster) {
		return false
	}
	// Prefer most up to date replica.
	return left.StorageInfo.Replication.Delay < right.StorageInfo.Replication.Delay
}
