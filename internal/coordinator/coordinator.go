package coordinator

import (
	"errors"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/vshard"
	"github.com/shmel1k/qumomf/pkg/vshard/orchestrator"
)

var (
	ErrClusterAlreadyExist = errors.New("cluster with such name already registered")
)

type shutdownTask func()

type Coordinator struct {
	// clusters contains registered Tarantool clusters
	// which Qumomf observes.
	clusters map[string]*vshard.Cluster

	// shutdownQueue contains all shutdown tasks to be
	// executed when coordinator is going to exit.
	shutdownQueue []shutdownTask
}

func New() *Coordinator {
	return &Coordinator{
		clusters: make(map[string]*vshard.Cluster),
	}
}

func (c Coordinator) RegisterCluster(name string, cfg config.ClusterConfig, globalCfg *config.Config) error {
	if _, exist := c.clusters[name]; exist {
		return ErrClusterAlreadyExist
	}

	cluster := vshard.NewCluster(name, cfg)
	c.clusters[name] = cluster
	c.addShutdownTask(cluster.Shutdown)

	mon := orchestrator.NewMonitor(orchestrator.Config{
		RecoveryPollTime:  globalCfg.Qumomf.ClusterRecoveryTime,
		DiscoveryPollTime: globalCfg.Qumomf.ClusterDiscoveryTime,
	}, cluster)
	c.addShutdownTask(mon.Shutdown)

	elector := quorum.NewLagQuorum() // TODO: move to cluster specific config
	failover := orchestrator.NewSwapMasterFailover(cluster, elector)
	c.addShutdownTask(failover.Shutdown)

	analysisStream := mon.Serve()
	failover.Serve(analysisStream)

	return nil
}

func (c Coordinator) Shutdown() {
	for i := len(c.shutdownQueue) - 1; i >= 0; i-- {
		task := c.shutdownQueue[i]
		task()
	}
}

func (c Coordinator) addShutdownTask(task shutdownTask) {
	c.shutdownQueue = append(c.shutdownQueue, task)
}