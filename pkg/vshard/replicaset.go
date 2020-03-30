package vshard

import (
	"encoding/json"
	"fmt"
)

type ReplicaSetUUID string

type ReplicaSet struct {
	// UUID is an unique identifier of the replica set in the cluster.
	UUID ReplicaSetUUID `json:"uuid"`

	// MasterUUID is an if of current master in the replica set.
	MasterUUID InstanceUUID `json:"master_uuid"`

	// Instances contains replication statistics and storage info
	// for all instances in the replica set in regard to the current master.
	Instances []Instance `json:"instances"`
}

func (set ReplicaSet) String() string {
	j, _ := json.Marshal(set)
	return string(j)
}

func (set *ReplicaSet) Followers() []Instance {
	if len(set.Instances) == 0 {
		return []Instance{}
	}

	followers := make([]Instance, 0, len(set.Instances)-1)
	for _, inst := range set.Instances {
		// Storage info contains the replica status in its set.
		// If we could not get that info, fall back to the replication data from the master.
		if inst.StorageInfo.ReplicationStatus == StatusFollow {
			followers = append(followers, inst)
		} else if inst.Upstream != nil && inst.Upstream.Status == UpstreamFollow {
			followers = append(followers, inst)
		}
	}

	return followers
}

func (set *ReplicaSet) Master() (Instance, error) {
	for _, inst := range set.Instances {
		if inst.UUID == set.MasterUUID {
			return inst, nil
		}
	}

	return Instance{}, fmt.Errorf("replica set `%s` has invalid topology snapshot: master `%s` not found", set.UUID, set.MasterUUID)
}
