package vshard

import "sync"

type ReplicaSet interface {
	GetShardUUID() ShardUUID
	GetConnectors() map[ReplicaUUID]*Connector

	GetMaster() ReplicaUUID
	SetMaster(ReplicaUUID)
}

type replicaset struct {
	mu         sync.RWMutex
	shardUUID  ShardUUID
	masterUUID ReplicaUUID
	connectors map[ReplicaUUID]*Connector
}

func NewReplicaSet(shardUUID ShardUUID, conns []*Connector) ReplicaSet {
	mp := make(map[ReplicaUUID]*Connector, len(conns))

	var master ReplicaUUID
	for _, v := range conns {
		uuid := ReplicaUUID(v.cfg.UUID)
		mp[uuid] = v
		if v.cfg.Master {
			master = uuid
		}
	}

	return &replicaset{
		shardUUID:  shardUUID,
		masterUUID: master,
		connectors: mp,
	}
}

func (r *replicaset) GetShardUUID() ShardUUID {
	return r.shardUUID
}

func (r *replicaset) GetMaster() ReplicaUUID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.masterUUID
}

func (r *replicaset) SetMaster(id ReplicaUUID) {
	r.mu.Lock()
	r.masterUUID = id
	r.mu.Unlock()
}

func (r *replicaset) GetConnectors() map[ReplicaUUID]*Connector {
	return r.connectors
}
