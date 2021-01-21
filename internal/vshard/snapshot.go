package vshard

// Snapshot is a copy of the cluster topology in given time.
type Snapshot struct {
	Created     int64        `json:"created"`
	Routers     []Router     `json:"routers"`
	ReplicaSets []ReplicaSet `json:"replica_sets"`
	priorities  map[string]int
}

func (s *Snapshot) ClusterHealthLevel() HealthLevel {
	hc := HealthCodeGreen
	for _, replicaSet := range s.ReplicaSets {
		gotHC, _ := replicaSet.HealthStatus()
		if gotHC > hc {
			hc = gotHC
		}
	}

	return s.healthLevel(hc)
}

func (s *Snapshot) healthLevel(healthCode HealthCode) HealthLevel {
	switch healthCode {
	case HealthCodeGreen:
		return HealthLevelGreen
	case HealthCodeYellow:
		return HealthLevelYellow
	case HealthCodeOrange:
		return HealthLevelOrange
	case HealthCodeRed:
		return HealthLevelRed
	}

	return HealthLevelUnknown
}

func (s *Snapshot) Copy() Snapshot {
	dst := Snapshot{
		Created:     s.Created,
		Routers:     make([]Router, len(s.Routers)),
		ReplicaSets: make([]ReplicaSet, 0, len(s.ReplicaSets)),
		priorities:  make(map[string]int),
	}

	for key, value := range s.priorities {
		dst.priorities[key] = value
	}

	for _, set := range s.ReplicaSets {
		dst.ReplicaSets = append(dst.ReplicaSets, set.Copy())
	}

	copy(dst.Routers, s.Routers)

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
