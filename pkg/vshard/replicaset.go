package vshard

import (
	"sync"
)

type Replicaset interface {
	GetMaster() string
	SetMaster(string)
	GetReplicas() map[string]*Connector
}

type replicaset struct {
	mu            sync.Mutex
	shards        map[string]*Connector
	currentMaster string
}

func NewReplicaset(conns []*Connector) Replicaset {
	mp := make(map[string]*Connector)

	master := ""
	for _, v := range conns {
		mp[v.cfg.UUID] = v
		if v.cfg.Master {
			master = v.cfg.UUID
		}
	}

	return &replicaset{
		currentMaster: master,
		shards:        mp,
	}
}

func (r *replicaset) GetMaster() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.currentMaster
}

func (r *replicaset) SetMaster(id string) {
	r.mu.Lock()
	r.currentMaster = id
	r.mu.Unlock()
}

func (r *replicaset) GetReplicas() map[string]*Connector {
	return r.shards
}
