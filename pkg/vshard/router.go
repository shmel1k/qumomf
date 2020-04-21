package vshard

type RouterUUID string
type InstanceStatus string

const (
	InstanceAvailable   InstanceStatus = "available"
	InstanceUnreachable InstanceStatus = "unreachable"
	InstanceMissing     InstanceStatus = "missing"
)

type Router struct {
	URI  string     `json:"uri"`
	UUID RouterUUID `json:"uuid"`
	Info RouterInfo `json:"info"`
}

func NewRouter(uri string, uuid RouterUUID) Router {
	return Router{
		URI:  uri,
		UUID: uuid,
		Info: RouterInfo{
			Status: -1,
		},
	}
}

type RouterInfo struct {
	LastSeen    int64                      `json:"last_seen"`
	ReplicaSets RouterReplicaSetParameters `json:"replica_sets"`
	Bucket      RouterBucket               `json:"bucket"`
	Status      int64                      `json:"status"`
	Alerts      []Alert                    `json:"alerts"`
}

type RouterReplicaSetParameters map[ReplicaSetUUID]RouterInstanceParameters

type RouterInstanceParameters struct {
	UUID           InstanceUUID   `json:"uuid"`
	Status         InstanceStatus `json:"status"`
	URI            string         `json:"uri"`
	NetworkTimeout float64        `json:"network_timeout"`
}

// RouterBucket represents bucket parameters known to the router.
type RouterBucket struct {
	// AvailableRO is the number of buckets known to the router
	// and available for read requests.
	AvailableRO int64 `json:"available_ro"`

	// AvailableRW is the number of buckets known to the router
	// and available for read and write requests.
	AvailableRW int64 `json:"available_rw"`

	// Unknown is the number of buckets known to the router
	// but unavailable for any requests.
	Unknown int64 `json:"unknown"`

	// Unreachable is the number of buckets
	// whose replica sets are not known to the router.
	Unreachable int64 `json:"unreachable"`
}
