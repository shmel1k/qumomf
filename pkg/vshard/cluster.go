package vshard

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/viciious/go-tarantool"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/pkg/util"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	vshardRouterInfoQuery = &tarantool.Call{
		Name: "vshard.router.info",
	}
	vshardBoxInfoQuery = &tarantool.Eval{
		// https://github.com/viciious/go-tarantool/issues/41
		Expression: `
			local repl = {}
			for i, r in pairs(box.info.replication) do
				if r.downstream ~= nil then
					r.downstream.vclock = string.format("%s", r.downstream.vclock)
				end
				repl[r.id] = r
			end
			return repl
		`,
	}
	vshardInstanceInfoQuery = &tarantool.Eval{
		Expression: `
			local data = {}
			data.storage = vshard.storage.info()
			data.read_only = box.cfg.read_only
			return data
		`,
	}
)

var (
	ErrMasterNotAvailable = errors.New("master of the replica set is not available so its topology could not be discovered")
	ErrReplicaSetNotFound = errors.New("replica set not found")
	ErrInstanceNotFound   = errors.New("instance not found")
)

type Cluster struct {
	Name string

	pool     ConnPool
	snapshot Snapshot

	readOnly          bool
	hasActiveRecovery bool

	mutex  sync.RWMutex
	logger zerolog.Logger
}

type Snapshot struct {
	Created     int64        `json:"created"`
	Routers     []Router     `json:"routers"`
	ReplicaSets []ReplicaSet `json:"replica_sets"`
}

func (s Snapshot) String() string {
	j, _ := json.Marshal(s)
	return string(j)
}

func (s *Snapshot) Copy() Snapshot {
	dst := Snapshot{
		Created:     s.Created,
		Routers:     make([]Router, len(s.Routers)),
		ReplicaSets: make([]ReplicaSet, len(s.ReplicaSets)),
	}

	copy(dst.Routers, s.Routers)
	copy(dst.ReplicaSets, s.ReplicaSets)

	return dst
}

func (s *Snapshot) TopologyOf(uuid ReplicaSetUUID) ([]Instance, error) {
	for _, set := range s.ReplicaSets {
		if set.UUID == uuid {
			return set.Instances, nil
		}
	}

	return []Instance{}, ErrReplicaSetNotFound
}

func NewCluster(name string, cfg config.ClusterConfig) *Cluster {
	connTemplate := ConnOptions{
		User:           *cfg.Connection.User,
		Password:       *cfg.Connection.Password,
		ConnectTimeout: *cfg.Connection.ConnectTimeout,
		QueryTimeout:   *cfg.Connection.RequestTimeout,
	}

	c := &Cluster{
		Name: name,
		pool: NewConnPool(connTemplate, cfg.OverrideURIRules),
		snapshot: Snapshot{
			Created: util.Timestamp(),
		},
		readOnly: *cfg.ReadOnly,
	}

	routers := make([]Router, 0, len(cfg.Routers))
	for _, r := range cfg.Routers {
		uri := r.Addr
		uuid := RouterUUID(r.UUID)
		routers = append(routers, NewRouter(uri, uuid))
	}
	c.snapshot.Routers = routers

	c.SetLogger(zerolog.Nop())

	return c
}

func (c *Cluster) SetLogger(logger zerolog.Logger) {
	c.logger = logger
}

func (c *Cluster) ConnRouter(r *Router) *Connector {
	return c.pool.Get(r.URI, string(r.UUID))
}

func (c *Cluster) ConnInstance(inst *Instance) *Connector {
	return c.pool.Get(inst.URI, string(inst.UUID))
}

func (c *Cluster) LastDiscovered() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.snapshot.Created
}

// SetReadOnly sets or clears the readonly mode for the cluster.
func (c *Cluster) SetReadOnly(v bool) {
	c.mutex.Lock()
	c.readOnly = v
	c.mutex.Unlock()
}

// ReadOnly indicates whether qumomf can run a failover
// or should just observe the cluster topology.
func (c *Cluster) ReadOnly() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.readOnly
}

func (c *Cluster) Routers() []Router {
	c.mutex.RLock()
	dst := make([]Router, len(c.snapshot.Routers))
	copy(dst, c.snapshot.Routers)
	c.mutex.RUnlock()

	return dst
}

func (c *Cluster) ReplicaSets() []ReplicaSet {
	c.mutex.RLock()
	dst := make([]ReplicaSet, len(c.snapshot.ReplicaSets))
	copy(dst, c.snapshot.ReplicaSets)
	c.mutex.RUnlock()

	return dst
}

func (c *Cluster) ReplicaSet(uuid ReplicaSetUUID) (ReplicaSet, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, set := range c.snapshot.ReplicaSets {
		if set.UUID == uuid {
			return set, nil
		}
	}

	return ReplicaSet{}, ErrReplicaSetNotFound
}

func (c *Cluster) Instances() []Instance {
	c.mutex.RLock()
	res := make([]Instance, 0)
	for i := range c.snapshot.ReplicaSets {
		set := &c.snapshot.ReplicaSets[i]
		res = append(res, set.Instances...)
	}
	c.mutex.RUnlock()

	return res
}

func (c *Cluster) Instance(uuid InstanceUUID) (Instance, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for i := range c.snapshot.ReplicaSets {
		set := &c.snapshot.ReplicaSets[i]
		for j := range set.Instances {
			inst := &set.Instances[j]
			if inst.UUID == uuid {
				return *inst, nil
			}
		}
	}

	return Instance{}, ErrInstanceNotFound
}

func (c *Cluster) StartRecovery() {
	c.mutex.Lock()
	c.hasActiveRecovery = true
	c.mutex.Unlock()
}

func (c *Cluster) StopRecovery() {
	c.mutex.Lock()
	c.hasActiveRecovery = false
	c.mutex.Unlock()
}

// HasActiveRecovery indicates when the cluster is suffering from
// some kind of failure and qumomf is running a failover process.
func (c *Cluster) HasActiveRecovery() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.hasActiveRecovery
}

func (c *Cluster) Shutdown() {
	c.pool.Close()
}

func (c *Cluster) Discover() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // TODO: move to config
	defer cancel()

	// Copy the current cluster snapshot to use it during the discovery.
	// It allows to discover and update the cluster state in a parallel way.
	c.mutex.RLock()
	snapshot := c.snapshot.Copy()
	c.mutex.RUnlock()

	router := pickUpRandomRouter(snapshot.Routers)
	if router == nil {
		c.logger.Error().Msg("There is no router in the cluster to discover its topology")
		return
	}
	c.logger.Debug().Msgf("Picked up the router '%s' in the cluster to discover its topology", router.UUID)

	// Read the topology configuration from the selected router.
	conn := c.pool.Get(router.URI, string(router.UUID))
	resp := conn.Exec(ctx, vshardRouterInfoQuery)
	if resp.Error != nil {
		c.logger.Err(resp.Error).Msgf("Failed to discover the topology of the cluster. Error code: %d", resp.ErrorCode)
		return
	}

	updatedRI, err := ParseRouterInfo(resp.Data)
	if err != nil {
		c.logger.Err(err).Msgf("Failed to discover the topology of the cluster using router '%s'", router.UUID)
		return
	}
	updatedRI.LastSeen = util.Timestamp()

	// Poll each instance of the cluster and collect the information.
	discovered := make(chan ReplicaSet, len(updatedRI.ReplicaSets))

	var wg sync.WaitGroup
	for setUUID, master := range updatedRI.ReplicaSets {
		wg.Add(1)

		go func(uuid ReplicaSetUUID, master RouterInstanceParameters) {
			defer wg.Done()

			topology, err := c.discoverReplication(ctx, master)
			if err != nil {
				c.logger.Err(err).
					Str("ReplicaSet", string(uuid)).
					Msg("Failed to update the topology, will use the previous snapshot")

				// Fallback to the previous snapshot data.
				topology, err = snapshot.TopologyOf(uuid)
				if err == ErrReplicaSetNotFound {
					c.logger.Error().
						Str("ReplicaSet", string(uuid)).
						Msg("There is no any previous snapshots of the topology")
					return
				}
			}

			c.discoverInstances(ctx, topology)

			set := ReplicaSet{
				UUID:       uuid,
				MasterUUID: master.UUID,
				Instances:  topology,
			}

			discovered <- set
		}(setUUID, master)
	}
	wg.Wait()

	close(discovered)

	ns := Snapshot{
		Created:     util.Timestamp(),
		Routers:     snapshot.Routers,
		ReplicaSets: make([]ReplicaSet, 0, len(discovered)),
	}
	for i := range ns.Routers {
		r := &ns.Routers[i]
		if r.UUID == router.UUID {
			r.Info = updatedRI
			break
		}
	}
	for set := range discovered {
		ns.ReplicaSets = append(ns.ReplicaSets, set)
	}

	c.logger.Debug().Msgf("Snapshot of the cluster has been created: %s", ns)

	c.mutex.Lock()
	if c.snapshot.Created <= ns.Created {
		c.snapshot = ns
	}
	c.mutex.Unlock()
}

func (c *Cluster) discoverReplication(ctx context.Context, master RouterInstanceParameters) ([]Instance, error) {
	if master.Status != InstanceAvailable {
		return []Instance{}, ErrMasterNotAvailable
	}

	conn := c.pool.Get(master.URI, string(master.UUID))
	resp := conn.Exec(ctx, vshardBoxInfoQuery)
	if resp.Error != nil {
		return []Instance{}, resp.Error
	}
	topology, err := ParseReplication(resp.Data)
	if err != nil {
		return []Instance{}, err
	}

	for i := 0; i < len(topology); i++ {
		inst := &topology[i]
		if inst.UUID == master.UUID {
			// We have to manually set the URI of the master,
			// because the master does not have an upstream data.
			inst.URI = master.URI
			break
		}
	}

	return topology, nil
}

func (c *Cluster) discoverInstances(ctx context.Context, instances []Instance) {
	var wg sync.WaitGroup
	for i := 0; i < len(instances); i++ {
		wg.Add(1)

		inst := &instances[i]
		go func() {
			c.discoverInstance(ctx, inst)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (c *Cluster) discoverInstance(ctx context.Context, inst *Instance) {
	conn := c.ConnInstance(inst)
	resp := conn.Exec(ctx, vshardInstanceInfoQuery)
	if resp.Error != nil {
		c.logger.Err(resp.Error).Msgf("Failed to discover the instance '%s'", inst.UUID)
		inst.LastCheckValid = false
		return
	}

	info, err := ParseInstanceInfo(resp.Data)
	if err != nil {
		c.logger.Err(err).Msgf("Failed to read info of the instance '%s'", inst.UUID)
		inst.LastCheckValid = false // TODO: not accurate
		return
	}

	inst.Readonly = info.Readonly
	inst.StorageInfo = info.StorageInfo
	inst.LastCheckValid = true
}

// pickUpRandomRouter selects a random router amongst provided ones.
func pickUpRandomRouter(routers []Router) *Router {
	if len(routers) == 0 {
		return nil
	}

	router := routers[rand.Intn(len(routers))]
	return &router
}
