package vshard

import (
	"sync/atomic"

	"github.com/shmel1k/qumomf/internal/config"
)

type Cluster interface {
	Name() string

	// SetReadOnly sets or clears the readonly mode for cluster.
	SetReadOnly(v bool)

	// ReadOnly indicates whether or not topology of the cluster
	// might be changed by orchestrator even in case of failure.
	ReadOnly() bool

	GetRouterConnectors() map[RouterUUID]*Connector
	GetReplicaSets() []ReplicaSet

	StartRecovery()
	StopRecovery()

	// HasActiveRecovery indicates when cluster is suffering from
	// some kind of failure and orchestrator run a failover process.
	HasActiveRecovery() bool

	Shutdown()
}

type cluster struct {
	name string

	routers     map[RouterUUID]*Connector
	replicaSets []ReplicaSet

	hasActiveRecovery atomic.Value
	readOnly          atomic.Value
}

func NewCluster(name string, cfg config.ClusterConfig) Cluster {
	c := &cluster{
		name:        name,
		routers:     make(map[RouterUUID]*Connector, len(cfg.Routers)),
		replicaSets: make([]ReplicaSet, 0, len(cfg.Shards)),
	}

	for uuid, instances := range cfg.Shards {
		connectors := make(map[ReplicaUUID]*Connector, len(instances))

		var masterUUID ReplicaUUID
		for _, inst := range instances {
			replicaUUID := ReplicaUUID(inst.UUID)
			conn := setupConnection(inst)
			connectors[replicaUUID] = conn
			if inst.Master {
				masterUUID = replicaUUID
			}
		}
		c.replicaSets = append(c.replicaSets, NewReplicaSet(ShardUUID(uuid), masterUUID, connectors))
	}

	for _, r := range cfg.Routers {
		uuid := RouterUUID(r.UUID)
		c.routers[uuid] = setupConnection(r)
	}

	if *cfg.ReadOnly {
		c.SetReadOnly(true)
	}

	return c
}

func (c *cluster) Name() string {
	return c.name
}

func (c *cluster) SetReadOnly(v bool) {
	if v {
		c.readOnly.Store(1)
	} else {
		c.readOnly.Store(0)
	}
}

func (c *cluster) ReadOnly() bool {
	return c.readOnly.Load() == 1
}

func (c *cluster) GetRouterConnectors() map[RouterUUID]*Connector {
	return c.routers
}

func (c *cluster) GetReplicaSets() []ReplicaSet {
	return c.replicaSets
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

	for _, set := range c.replicaSets {
		for _, conn := range set.GetConnectors() {
			conn.Close()
		}
	}
}
