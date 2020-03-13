package orchestrator

import "fmt"

type bucket struct {
	Active    uint64
	Garbage   int64
	Pinned    int64
	Receiving int64
	Sending   int64
	Total     uint64
}

type set struct {
	UUID   string
	Master string
}

type replicasets map[string]set

type replication struct {
	Status string
	Lag    float64
}

type storageInfo struct {
	Alerts      []interface{}
	Bucket      *bucket
	Replicasets replicasets
	Replication replication
}

func parseBucket(dt map[string]interface{}) (*bucket, error) {
	mp, ok := dt["bucket"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to cast bucket field %v %T to map[string]interface{}", dt["bucket"], dt["bucket"])
	}

	active, ok := mp["active"].(uint64)
	if !ok {
		return nil, fmt.Errorf("failed to case 'active' field %v %T to int64", mp["active"], mp["active"])
	}

	garbage, ok := mp["garbage"].(int64)
	if !ok {
		return nil, fmt.Errorf("failed to case 'garbage' field %v %T to int64", mp["garbage"], mp["garbage"])
	}

	pinned, ok := mp["pinned"].(int64)
	if !ok {
		return nil, fmt.Errorf("failed to case 'pinned' field %v %T to int64", mp["pinned"], mp["pinned"])
	}

	receiving, ok := mp["receiving"].(int64)
	if !ok {
		return nil, fmt.Errorf("failed to case 'receiving' field %v %T to int64", mp["receiving"], mp["receiving"])
	}

	sending, ok := mp["sending"].(int64)
	if !ok {
		return nil, fmt.Errorf("failed to case 'sending' field %v %T to int64", mp["sending"], mp["sending"])
	}

	total, ok := mp["total"].(uint64)
	if !ok {
		return nil, fmt.Errorf("failed to case 'total' field %v %T to int64", mp["total"], mp["total"])
	}

	return &bucket{
		Active:    active,
		Garbage:   garbage,
		Pinned:    pinned,
		Receiving: receiving,
		Sending:   sending,
		Total:     total,
	}, nil
}

func parseSet(dt map[string]interface{}) (set, error) {
	master, ok := dt["master"].(map[string]interface{})["uri"].(string)
	if !ok {
		return set{}, fmt.Errorf("failed to parse 'master' field from replicaset %v", dt)
	}

	uuid, ok := dt["uuid"].(string)
	if !ok {
		return set{}, fmt.Errorf("failed to parse 'uuid' field from replicaset %v", dt)
	}

	return set{
		Master: master,
		UUID:   uuid,
	}, nil
}

func parseReplicasets(dt map[string]interface{}) (replicasets, error) {
	result := make(replicasets)

	for k, v := range dt {
		var err error
		result[k], err = parseSet(v.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func parseReplication(dt map[string]interface{}) (replication, error) {
	status, ok := dt["status"].(string)
	if !ok {
		return replication{}, fmt.Errorf("failed to parse field 'status' from replication %v", dt)
	}
	if lag, ok := dt["lag"].(float64); ok {
		return replication{
			Status: status,
			Lag:    lag,
		}, nil
	}

	return replication{
		Status: status,
	}, nil
}

func parseStorageInfo(data [][]interface{}) (*storageInfo, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("got empty response %v from tarantool", data)
	}

	data1 := data[0]
	if len(data1) == 0 {
		return nil, fmt.Errorf("got empty data %v from tarantool", data1)
	}

	dt, ok := data1[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to case %v %T to map[string]interface{}", data1[0], data1[0])
	}

	alerts, ok := dt["alerts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no alerts field found in data %v", data1)
	}

	b, err := parseBucket(dt)
	if err != nil {
		return nil, err
	}

	sets, err := parseReplicasets(dt["replicasets"].(map[string]interface{}))
	if err != nil {
		return nil, err
	}

	repl, err := parseReplication(dt["replication"].(map[string]interface{}))
	if err != nil {
		return nil, err
	}

	return &storageInfo{
		Alerts:      alerts,
		Bucket:      b,
		Replicasets: sets,
		Replication: repl,
	}, nil
}
