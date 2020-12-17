package vshard

import (
	"fmt"
	"strconv"
	"strings"
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

func (set ReplicaSet) HealthStatus() (code HealthCode, level HealthLevel) {
	master, err := set.Master()
	if err != nil {
		return HealthCodeUnknown, HealthLevelUnknown
	}

	return master.CriticalCode(), master.CriticalLevel()
}

func (set ReplicaSet) Followers() []Instance {
	if len(set.Instances) == 0 {
		return []Instance{}
	}

	followers := make([]Instance, 0, len(set.Instances)-1)
	for _, inst := range set.Instances { //nolint:gocritic
		if inst.UUID != set.MasterUUID {
			followers = append(followers, inst)
		}
	}

	return followers
}

func (set ReplicaSet) AliveFollowers() []Instance {
	if len(set.Instances) == 0 {
		return []Instance{}
	}

	followers := make([]Instance, 0, len(set.Instances)-1)
	for _, inst := range set.Instances { // nolint:gocritic
		if inst.UUID == set.MasterUUID {
			continue
		}

		if !inst.LastCheckValid {
			continue
		}

		upstream := inst.Upstream
		downstream := inst.Downstream

		if upstream == nil && downstream == nil {
			continue
		}

		if upstream != nil {
			if upstream.Status != UpstreamDisconnected && upstream.Status != UpstreamStopped {
				followers = append(followers, inst)
			}
		} else if downstream != nil {
			if downstream.Status != DownstreamStopped {
				followers = append(followers, inst)
			}
		}
	}

	return followers
}

func (set ReplicaSet) Master() (Instance, error) {
	for i := range set.Instances {
		inst := &set.Instances[i]
		if inst.UUID == set.MasterUUID {
			return *inst, nil
		}
	}

	return Instance{}, fmt.Errorf("replica set `%s` has invalid topology snapshot: master `%s` not found", set.UUID, set.MasterUUID)
}

func (set ReplicaSet) String() string {
	// Minimal style, only important info.
	master, err := set.Master()
	if err != nil {
		return fmt.Sprintf("failed to get replica set master: %s", err)
	}

	var sb strings.Builder
	sb.WriteString("id: ")
	sb.WriteString(string(set.UUID))
	sb.WriteString("; master uuid: ")
	sb.WriteString(string(set.MasterUUID))
	sb.WriteString("; master uri: ")
	sb.WriteString(master.URI)
	sb.WriteString("; size: ")
	sb.WriteString(strconv.Itoa(len(set.Instances)))
	sb.WriteString("; health: ")
	_, cl := set.HealthStatus()
	sb.WriteString(string(cl))

	if cl == HealthLevelGreen {
		return sb.String()
	}

	sb.WriteString("; alerts: [")
	prettyList := false
	for i := range set.Instances {
		inst := &set.Instances[i]
		alerts := inst.StorageInfo.Alerts
		if len(alerts) > 0 {
			if prettyList {
				sb.WriteString(", ")
			}
			sb.WriteString(string(inst.UUID))
			sb.WriteString(" -> ")
			for j, alert := range alerts {
				sb.WriteString(alert.String())
				if j != len(alerts)-1 {
					sb.WriteString(", ")
				}
			}
			prettyList = true
		}
	}
	sb.WriteString("]")

	return sb.String()
}
