package vshard

import "sync/atomic"

type Cluster interface {
	GetRouterConnectors() map[RouterUUID]*Connector
	GetReplicaSets() []ReplicaSet

	StartRecovery()
	StopRecovery()
	HasActiveRecovery() bool

	Shutdown()
}

type cluster struct {
	routers           map[RouterUUID]*Connector
	replicas          []ReplicaSet
	hasActiveRecovery atomic.Value
}

func NewCluster(routers []InstanceConfig, sets map[ShardUUID][]InstanceConfig) Cluster {
	res := &cluster{
		routers:  make(map[RouterUUID]*Connector, len(routers)),
		replicas: make([]ReplicaSet, 0, len(sets)),
	}

	for uuid, v := range sets {
		conns := make([]*Connector, 0, len(v))
		for j := range v {
			conn := setupConnection(&sets[uuid][j])
			conns = append(conns, conn)
		}
		res.replicas = append(res.replicas, NewReplicaSet(uuid, conns))
	}

	for i, r := range routers {
		uuid := RouterUUID(r.UUID)
		res.routers[uuid] = setupConnection(&routers[i])
	}

	return res
}

func (c *cluster) GetRouterConnectors() map[RouterUUID]*Connector {
	return c.routers
}

func (c *cluster) GetReplicaSets() []ReplicaSet {
	return c.replicas
}

func (c *cluster) StartRecovery() {
	c.hasActiveRecovery.Store(1)
}

func (c *cluster) StopRecovery() {
	c.hasActiveRecovery.Store(0)
}

func (c *cluster) HasActiveRecovery() bool {
	return c.hasActiveRecovery.Load() == 1
}

func (c *cluster) Shutdown() {
	for _, conn := range c.routers {
		conn.Close()
	}

	for _, set := range c.replicas {
		for _, conn := range set.GetConnectors() {
			conn.Close()
		}
	}
}
