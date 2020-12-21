package vshard

// Snapshot is a copy of the cluster topology in given time.
type Snapshot struct {
	Created     int64        `json:"created"`
	Routers     []Router     `json:"routers"`
	ReplicaSets []ReplicaSet `json:"replica_sets"`
	priorities  map[string]int
}

func (s *Snapshot) Copy() Snapshot {
	dst := Snapshot{
		Created:     s.Created,
		Routers:     make([]Router, len(s.Routers)),
		ReplicaSets: make([]ReplicaSet, len(s.ReplicaSets)),
		priorities:  s.priorities,
	}

	copy(dst.Routers, s.Routers)
	copy(dst.ReplicaSets, s.ReplicaSets)

	return dst
}

func (s *Snapshot) TopologyOf(uuid ReplicaSetUUID) ([]Instance, error) {
	for _, set := range s.ReplicaSets {
		if set.UUID == uuid {
			return set.Instances, nil
		}
	}

	return []Instance{}, ErrReplicaSetNotFound
}

func (s *Snapshot) ReplicaSet(uuid ReplicaSetUUID) (ReplicaSet, error) {
	for _, set := range s.ReplicaSets {
		if set.UUID == uuid {
			return set, nil
		}
	}

	return ReplicaSet{}, ErrReplicaSetNotFound
}

func (s *Snapshot) UpdatePriorities(priorities map[string]int) {
	s.priorities = priorities

	for i := range s.ReplicaSets {
		set := &s.ReplicaSets[i]
		for j := range set.Instances {
			inst := &set.Instances[j]
			if priority, ok := s.priorities[string(inst.UUID)]; ok {
				inst.Priority = priority
			} else {
				inst.Priority = 0
			}
		}
	}
}
