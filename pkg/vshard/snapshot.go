package vshard

// Snapshot is a copy of the cluster topology in given time.
type Snapshot struct {
	Created     int64        `json:"created"`
	Routers     []Router     `json:"routers"`
	ReplicaSets []ReplicaSet `json:"replica_sets"`
}

func (s *Snapshot) Copy() Snapshot {
	dst := Snapshot{
		Created:     s.Created,
		Routers:     make([]Router, len(s.Routers)),
		ReplicaSets: make([]ReplicaSet, len(s.ReplicaSets)),
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
