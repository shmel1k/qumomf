package vshard

type Cluster interface {
	GetReplicasets() []Replicaset
}

type cluster struct {
	replicas []Replicaset
}

func NewCluster(c [][]InstanceConfig) Cluster {
	res := &cluster{
		replicas: make([]Replicaset, 0, len(c)),
	}

	for _, v := range c {
		conns := make([]*Connector, 0, len(v))
		for _, vv := range v {
			conn := setupConnection(vv)
			conns = append(conns, conn)
		}
		res.replicas = append(res.replicas, NewReplicaset(conns))
	}

	return res
}

func (c *cluster) GetReplicasets() []Replicaset {
	return c.replicas
}
