package vshard

type InstanceUUID string
type ReplicationStatus string
type UpstreamStatus string

const (
	StatusFollow ReplicationStatus = "follow"
	StatusMaster ReplicationStatus = "master"
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

type Instance struct {
	// UUID is a global unique identifier of the instance.
	UUID InstanceUUID `json:"uuid"`

	// URI contains the replication user name, host IP address and port number used for the instance.
	URI string `json:"uri"`

	// LastCheckValid indicates whether the last check of the instance by qumomf was successful or not.
	LastCheckValid bool `json:"last_check_valid"`

	// LSN is the log sequence number (LSN) for the latest entry in the instance’s write ahead log (WAL).
	LSN int64 `json:"lsn"`

	// Upstream contains statistics for the replication data uploaded by the instance.
	Upstream *Upstream `json:"upstream"`

	// StorageInfo contains the information about the storage instance.
	StorageInfo StorageInfo `json:"storage_info"`
}

// Upstream contains statistics for the replication data uploaded by the instance.
type Upstream struct {
	// Peer contains the replication user name, host IP address and port number used for the instance.
	Peer string

	//  Status is the replication status of the instance.
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

func (i *Instance) HasAlert(t AlertType) bool {
	for _, a := range i.StorageInfo.Alerts {
		if a.Type == t {
			return true
		}
	}

	return false
}

type StorageInfo struct {
	Bucket            InstanceBucket    `json:"bucket"`
	ReplicationStatus ReplicationStatus `json:"replication_status"`
	Alerts            []Alert           `json:"alerts"`
}

type InstanceBucket struct {
	Active    int64 `json:"active"`
	Garbage   int64 `json:"garbage"`
	Pinned    int64 `json:"pinned"`
	Receiving int64 `json:"receiving"`
	Sending   int64 `json:"sending"`
	Total     int64 `json:"total"`
}
