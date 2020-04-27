package vshard

import (
	"fmt"
)

var (
	ErrEmptyResponse     = fmt.Errorf("got empty response from tarantool")
	ErrNoRouterInfo      = fmt.Errorf("got empty router info from tarantool")
	ErrNoInstanceInfo    = fmt.Errorf("got empty instance info from tarantool")
	ErrNoReplicationInfo = fmt.Errorf("got empty replicaction info from tarantool")
)

type container map[string]interface{}

func castToContainer(src interface{}) (container, error) {
	dt, ok := src.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to cast %T to map[string]interface{}, data: '%v'", src, src)
	}

	return dt, nil
}

func (c container) getInt64(key string) (int64, error) {
	switch t := c[key].(type) {
	case int64:
		return t, nil
	case uint64:
		return int64(t), nil
	default:
		return 0, fmt.Errorf("field '%s' (%T) is not found or has unexpected type in container: %v", key, c[key], c)
	}
}

func (c container) getUInt64(key string) (uint64, error) {
	switch t := c[key].(type) {
	case int64:
		return uint64(t), nil
	case uint64:
		return t, nil
	default:
		return 0, fmt.Errorf("field '%s' (%T) is not found or has unexpected type in container: %v", key, c[key], c)
	}
}

func (c container) getFloat64(key string) (float64, error) {
	switch t := c[key].(type) {
	case float64:
		return t, nil
	case uint64:
		return float64(t), nil
	case int64:
		return float64(t), nil
	default:
		return 0, fmt.Errorf("field '%s' (%T) is not found or has unexpected type in container: %v", key, c[key], c)
	}
}

func (c container) getArray(key string) ([]interface{}, error) {
	v, ok := c[key].([]interface{})
	if !ok {
		return nil, fmt.Errorf("field '%s' (%T) is not found or has unexpected type in container: %v", key, c[key], c)
	}
	return v, nil
}

func (c container) getContainer(key string) (container, error) {
	v, err := castToContainer(c[key])
	if err != nil {
		return nil, fmt.Errorf("field '%s' is not found or has unexpected type in container: %v", key, c)
	}
	return v, nil
}

func (c container) getString(key string) (string, error) {
	v, ok := c[key].(string)
	if !ok {
		return "", fmt.Errorf("field '%s' (%T) is not found or has unexpected type in container: %v", key, c[key], c)
	}
	return v, nil
}

func (c container) getBool(key string) (bool, error) {
	v, ok := c[key].(bool)
	if !ok {
		return false, fmt.Errorf("field '%s' (%T) is not found or has unexpected type in container: %v", key, c[key], c)
	}
	return v, nil
}

func ParseRouterInfo(data [][]interface{}) (RouterInfo, error) {
	if len(data) == 0 {
		return RouterInfo{}, ErrEmptyResponse
	}

	tuple := data[0]
	if len(tuple) == 0 {
		return RouterInfo{}, ErrNoRouterInfo
	}

	dt, err := castToContainer(tuple[0])
	if err != nil {
		return RouterInfo{}, err
	}

	alerts, err := parseAlerts(dt)
	if err != nil {
		return RouterInfo{}, err
	}

	status, err := dt.getInt64("status")
	if err != nil {
		return RouterInfo{}, err
	}

	bucket, err := parseRouterBucket(dt)
	if err != nil {
		return RouterInfo{}, err
	}

	sets, err := parseRouterReplicaSets(dt)
	if err != nil {
		return RouterInfo{}, err
	}

	return RouterInfo{
		Bucket:      bucket,
		Status:      status,
		Alerts:      alerts,
		ReplicaSets: sets,
	}, nil
}

func parseRouterReplicaSets(dt container) (RouterReplicaSetParameters, error) {
	mp, err := dt.getContainer("replicasets")
	if err != nil {
		return nil, err
	}

	result := make(RouterReplicaSetParameters)

	for uuid, v := range mp {
		vc, err := castToContainer(v)
		if err != nil {
			return nil, err
		}

		result[ReplicaSetUUID(uuid)], err = parseRouterInstance(vc)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func parseRouterInstance(dt container) (RouterInstanceParameters, error) {
	mp, err := dt.getContainer("master")
	if err != nil {
		return RouterInstanceParameters{}, err
	}

	uuid, err := mp.getString("uuid")
	if err != nil {
		return RouterInstanceParameters{}, err
	}

	uri, err := mp.getString("uri")
	if err != nil {
		return RouterInstanceParameters{}, err
	}

	status, err := mp.getString("status")
	if err != nil {
		return RouterInstanceParameters{}, err
	}

	timeout, err := mp.getFloat64("network_timeout")
	if err != nil {
		return RouterInstanceParameters{}, err
	}

	return RouterInstanceParameters{
		UUID:           InstanceUUID(uuid),
		Status:         InstanceStatus(status),
		URI:            uri,
		NetworkTimeout: timeout,
	}, nil
}

func parseRouterBucket(dt container) (RouterBucket, error) {
	mp, err := dt.getContainer("bucket")
	if err != nil {
		return RouterBucket{}, err
	}

	availableRO, err := mp.getInt64("available_ro")
	if err != nil {
		return RouterBucket{}, err
	}

	availableRW, err := mp.getInt64("available_rw")
	if err != nil {
		return RouterBucket{}, err
	}

	unknown, err := mp.getInt64("unknown")
	if err != nil {
		return RouterBucket{}, err
	}

	unreachable, err := mp.getInt64("unreachable")
	if err != nil {
		return RouterBucket{}, err
	}

	return RouterBucket{
		AvailableRO: availableRO,
		AvailableRW: availableRW,
		Unknown:     unknown,
		Unreachable: unreachable,
	}, nil
}

func ParseInstanceInfo(data [][]interface{}) (InstanceInfo, error) {
	if len(data) == 0 {
		return InstanceInfo{}, ErrEmptyResponse
	}

	tuple := data[0]
	if len(tuple) == 0 {
		return InstanceInfo{}, ErrNoInstanceInfo
	}

	dt, err := castToContainer(tuple[0])
	if err != nil {
		return InstanceInfo{}, err
	}

	readonly, err := dt.getBool("read_only")
	if err != nil {
		return InstanceInfo{}, err
	}

	fingerprint, err := dt.getUInt64("vshard_fingerprint")
	if err != nil {
		return InstanceInfo{}, err
	}

	storageInfo, err := parseStorageInfo(dt)
	if err != nil {
		return InstanceInfo{}, err
	}

	return InstanceInfo{
		Readonly:          readonly,
		VShardFingerprint: fingerprint,
		StorageInfo:       storageInfo,
	}, nil
}

func parseStorageInfo(dt container) (StorageInfo, error) {
	_, ok := dt["storage"]
	if !ok {
		return StorageInfo{}, nil
	}

	s, err := dt.getContainer("storage")
	if err != nil {
		return StorageInfo{}, err
	}

	setStatus, err := s.getInt64("status")
	if err != nil {
		return StorageInfo{}, err
	}

	alerts, err := parseAlerts(s)
	if err != nil {
		return StorageInfo{}, err
	}

	replication, err := s.getContainer("replication")
	if err != nil {
		return StorageInfo{}, err
	}
	delay := float64(0)
	if _, ok := replication["lag"]; ok {
		delay, err = replication.getFloat64("lag")
		if err != nil {
			return StorageInfo{}, err
		}
	}
	if _, ok := replication["idle"]; ok {
		delay, err = replication.getFloat64("idle")
		if err != nil {
			return StorageInfo{}, err
		}
	}
	replStatus, err := replication.getString("status")
	if err != nil {
		return StorageInfo{}, err
	}

	bucket, err := parseInstanceBucket(s)
	if err != nil {
		return StorageInfo{}, err
	}

	return StorageInfo{
		Status: HealthCode(setStatus),
		Replication: Replication{
			Status: ReplicationStatus(replStatus),
			Delay:  delay,
		},
		Bucket: bucket,
		Alerts: alerts,
	}, nil
}

func parseInstanceBucket(dt container) (InstanceBucket, error) {
	mp, err := dt.getContainer("bucket")
	if err != nil {
		return InstanceBucket{}, err
	}

	active, err := mp.getInt64("active")
	if err != nil {
		return InstanceBucket{}, err
	}

	garbage, err := mp.getInt64("garbage")
	if err != nil {
		return InstanceBucket{}, err
	}

	pinned, err := mp.getInt64("pinned")
	if err != nil {
		return InstanceBucket{}, err
	}

	receiving, err := mp.getInt64("receiving")
	if err != nil {
		return InstanceBucket{}, err
	}

	sending, err := mp.getInt64("sending")
	if err != nil {
		return InstanceBucket{}, err
	}

	total, err := mp.getInt64("total")
	if err != nil {
		return InstanceBucket{}, err
	}

	return InstanceBucket{
		Active:    active,
		Garbage:   garbage,
		Pinned:    pinned,
		Receiving: receiving,
		Sending:   sending,
		Total:     total,
	}, nil
}

func parseAlerts(dt container) ([]Alert, error) {
	mp, err := dt.getArray("alerts")
	if err != nil {
		return nil, err
	}

	alerts := make([]Alert, 0, len(mp))
	for _, arr := range mp {
		arr, ok := arr.([]interface{})
		if !ok {
			return nil, fmt.Errorf("field alerts has unexpected type in container: %v", dt)
		}

		if len(arr) < 2 {
			continue
		}

		aType, ok := arr[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected to find strings in alert array, got %T, alert: %v", arr[0], arr)
		}

		aDesc, ok := arr[1].(string)
		if !ok {
			return nil, fmt.Errorf("expected to find strings in alert array, got %T, alert: %v", arr[0], arr)
		}

		alerts = append(alerts, Alert{
			Type:        AlertType(aType),
			Description: aDesc,
		})
	}

	return alerts, nil
}

func ParseReplication(data [][]interface{}) ([]Instance, error) {
	if len(data) == 0 {
		return nil, ErrEmptyResponse
	}

	tuple := data[0]
	if len(tuple) == 0 {
		return nil, ErrNoReplicationInfo
	}

	instances := make([]Instance, 0, len(tuple))
	for _, t := range tuple {
		mp, err := castToContainer(t)
		if err != nil {
			return nil, err
		}

		id, err := mp.getUInt64("id")
		if err != nil {
			return nil, err
		}

		uuid, err := mp.getString("uuid")
		if err != nil {
			return nil, err
		}

		lsn, err := mp.getInt64("lsn")
		if err != nil {
			return nil, err
		}

		lsnBehindMaster := int64(0)
		if _, ok := mp["lsn_behind_master"]; ok {
			lsnBehindMaster, err = mp.getInt64("lsn_behind_master")
			if err != nil {
				return nil, err
			}
		}

		upstream, err := parseUpstream(mp)
		if err != nil {
			return nil, err
		}

		downstream, err := parseDownstream(mp)
		if err != nil {
			return nil, err
		}

		uri := ""
		if upstream != nil {
			uri = upstream.Peer
		}

		inst := Instance{
			ID:              id,
			UUID:            InstanceUUID(uuid),
			URI:             uri,
			LSN:             lsn,
			LSNBehindMaster: lsnBehindMaster,
			Upstream:        upstream,
			Downstream:      downstream,
		}

		instances = append(instances, inst)
	}

	return instances, nil
}

func parseUpstream(dt container) (*Upstream, error) {
	_, ok := dt["upstream"]
	if !ok {
		return nil, nil
	}

	u, err := dt.getContainer("upstream")
	if err != nil {
		return nil, err
	}

	peer, err := u.getString("peer")
	if err != nil {
		return nil, err
	}

	status, err := u.getString("status")
	if err != nil {
		return nil, err
	}

	idle, err := u.getFloat64("idle")
	if err != nil {
		return nil, err
	}

	lag, err := u.getFloat64("lag")
	if err != nil {
		return nil, err
	}

	message := ""
	if _, ok := u["message"]; ok {
		message, err = u.getString("message")
		if err != nil {
			return nil, err
		}
	}

	return &Upstream{
		Peer:    peer,
		Status:  UpstreamStatus(status),
		Idle:    idle,
		Lag:     lag,
		Message: message,
	}, nil
}

func parseDownstream(dt container) (*Downstream, error) {
	_, ok := dt["downstream"]
	if !ok {
		return nil, nil
	}

	u, err := dt.getContainer("downstream")
	if err != nil {
		return nil, err
	}

	status, err := u.getString("status")
	if err != nil {
		return nil, err
	}

	return &Downstream{
		Status: DownstreamStatus(status),
	}, nil
}
