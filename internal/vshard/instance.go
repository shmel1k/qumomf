package vshard

import "strings"

type InstanceUUID string

type ReplicationStatus string
type UpstreamStatus string
type DownstreamStatus string

type HealthCode int
type HealthLevel string

const (
	StatusFollow       ReplicationStatus = "follow"
	StatusMaster       ReplicationStatus = "master"
	StatusDisconnected ReplicationStatus = "disconnected"
)

const (
	UpstreamAuth         UpstreamStatus = "auth"         // the instance is getting authenticated to connect to a replication source.
	UpstreamConnecting   UpstreamStatus = "connecting"   // the instance is trying to connect to the replications source(s) listed in its replication parameter.
	UpstreamDisconnected UpstreamStatus = "disconnected" // the instance is not connected to the replica set (due to network problems, not replication errors).
	UpstreamFollow       UpstreamStatus = "follow"       // the replication is in progress.
	UpstreamRunning      UpstreamStatus = "running"      // the instance’s role is “master” (non read-only) and replication is in progress.
	UpstreamStopped      UpstreamStatus = "stopped"      // the replication was stopped due to a replication error (e.g. duplicate key).
	UpstreamOrphan       UpstreamStatus = "orphan"       // the instance has not (yet) succeeded in joining the required number of masters (see orphan status).
	UpstreamSync         UpstreamStatus = "sync"         // the master and replica are synchronizing to have the same data.
)

const (
	DownstreamFollow  DownstreamStatus = "follow"  // the downstream replication is in progress.
	DownstreamStopped DownstreamStatus = "stopped" // the downstream replication has stopped.
)

const (
	// A replica set works in a regular way.
	HealthCodeGreen HealthCode = 0
	// There are some issues, but they don’t affect a replica set efficiency
	// (worth noticing, but don’t require immediate intervention).
	HealthCodeYellow HealthCode = 1
	// A replica set is in a degraded state.
	HealthCodeOrange HealthCode = 2
	// A replica set is disabled.
	HealthCodeRed HealthCode = 3
	// If something will change.
	HealthCodeUnknown HealthCode = 4
)

const (
	HealthLevelGreen   HealthLevel = "green"
	HealthLevelYellow  HealthLevel = "yellow"
	HealthLevelOrange  HealthLevel = "orange"
	HealthLevelRed     HealthLevel = "red"
	HealthLevelUnknown HealthLevel = "unknown" // if something will change
)

type Instance struct {
	// ID is a short numeric identifier of the instance within the replica set.
	ID uint64 `json:"id"`

	// UUID is a global unique identifier of the instance.
	UUID InstanceUUID `json:"uuid"`

	// URI contains the replication user name, host IP address and port number of the instance.
	URI string `json:"uri"`

	// Readonly indicates whether the instance is readonly or readwrite.
	Readonly bool `json:"readonly"`

	// LastCheckValid indicates whether the last check of the instance by qumomf was successful or not.
	LastCheckValid bool `json:"last_check_valid"`

	// LSN is the log sequence number (LSN) for the latest entry in the instance’s write ahead log (WAL).
	LSN int64 `json:"lsn"`

	// LSNBehindMaster is a measure of how the replica is far from master.
	LSNBehindMaster int64 `json:"lsn_behind_master"`

	// Upstream contains statistics for the replication data uploaded by the instance.
	Upstream *Upstream `json:"upstream"`

	// Downstream contains statistics for the replication data requested and downloaded from the instance.
	Downstream *Downstream `json:"downstream"`

	// StorageInfo contains the information about the storage instance.
	StorageInfo StorageInfo `json:"storage_info"`

	// VShardFingerprint is a CRC32 hash code of the shard topology configuration.
	VShardFingerprint uint64 `json:"vshard_fingerprint"`

	// Priority helps to choose the best candidate during the failover using
	// user promotion rules.
	//
	// If priority less than 0, instance will not participate in the master election.
	Priority int `json:"priority"`
}

// InstanceIdent contains unique UUID and URI of the instance.
type InstanceIdent struct {
	UUID InstanceUUID
	URI  string
}

func (ident InstanceIdent) String() string {
	var sb strings.Builder
	sb.Grow(len(ident.URI) + len(ident.UUID) + 1)
	sb.WriteString(string(ident.UUID))
	sb.WriteRune('/')
	sb.WriteString(ident.URI)

	return sb.String()
}

// Upstream contains statistics for the replication data uploaded by the instance.
type Upstream struct {
	// Peer contains the replication user name, host IP address and port number used for the instance.
	Peer string `json:"peer"`

	// Status is the replication status of the instance.
	Status UpstreamStatus `json:"status"`

	// Idle is the time (in seconds) since the instance received the last event from a master.
	// This is the primary indicator of replication health.
	Idle float64 `json:"idle"`

	// Lag is the time difference between the local time at the instance, recorded when the event was received,
	// and the local time at another master recorded when the event was written to the write ahead log on that master.
	Lag float64 `json:"lag"`

	// Message contains an error message in case of a degraded state, empty otherwise.
	Message string `json:"message"`
}

type Downstream struct {
	// Status is the replication status for downstream replications.
	Status DownstreamStatus `json:"status"`
}

func (i *Instance) Ident() InstanceIdent {
	return InstanceIdent{
		UUID: i.UUID,
		URI:  i.URI,
	}
}

func (i *Instance) HasAlert(t AlertType) bool {
	for _, a := range i.StorageInfo.Alerts {
		if a.Type == t {
			return true
		}
	}

	return false
}

func (i *Instance) CriticalCode() HealthCode {
	return i.StorageInfo.Status
}

func (i *Instance) CriticalLevel() HealthLevel {
	switch i.CriticalCode() {
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

// InstanceInfo is a helper structure contains
// instance info in custom format.
type InstanceInfo struct {
	Readonly          bool
	VShardFingerprint uint64
	StorageInfo       StorageInfo
}

type StorageInfo struct {
	// Status indicates current state of the ReplicaSet.
	// It ranges from 0 (green) up to 3 (red).
	Status      HealthCode     `json:"status"`
	Replication Replication    `json:"replication"`
	Bucket      InstanceBucket `json:"bucket"`
	Alerts      []Alert        `json:"alerts"`
}

type Replication struct {
	Status ReplicationStatus `json:"status"`

	// Delay might be the lag or idle depends on the replication status.
	// Tarantool returns idle when replication is broken otherwise the lag.
	Delay float64 `json:"delay"`
}

type InstanceBucket struct {
	Active    int64 `json:"active"`
	Garbage   int64 `json:"garbage"`
	Pinned    int64 `json:"pinned"`
	Receiving int64 `json:"receiving"`
	Sending   int64 `json:"sending"`
	Total     int64 `json:"total"`
}
