package coordinator

import (
	"errors"

	"github.com/rs/zerolog"

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
	logger zerolog.Logger

	// clusters contains registered Tarantool clusters
	// which Qumomf observes.
	clusters map[string]*vshard.Cluster

	// shutdownQueue contains all shutdown tasks to be
	// executed when coordinator is going to exit.
	shutdownQueue []shutdownTask
}

func New(logger zerolog.Logger) *Coordinator {
	return &Coordinator{
		logger:   logger,
		clusters: make(map[string]*vshard.Cluster),
	}
}

func (c Coordinator) RegisterCluster(name string, cfg config.ClusterConfig, globalCfg *config.Config) error {
	if _, exist := c.clusters[name]; exist {
		return ErrClusterAlreadyExist
	}

	clusterLogger := c.logger.With().Str("cluster", name).Logger()

	cluster := vshard.NewCluster(name, cfg)
	cluster.SetLogger(clusterLogger)
	c.clusters[name] = cluster
	c.addShutdownTask(cluster.Shutdown)

	mon := orchestrator.NewMonitor(cluster, orchestrator.Config{
		RecoveryPollTime:  globalCfg.Qumomf.ClusterRecoveryTime,
		DiscoveryPollTime: globalCfg.Qumomf.ClusterDiscoveryTime,
	}, clusterLogger)
	c.addShutdownTask(mon.Shutdown)

	elector := quorum.New(quorum.Mode(*cfg.ElectionMode))
	failover := orchestrator.NewDefaultFailover(cluster, orchestrator.FailoverConfig{
		Elector:                     elector,
		ReplicaSetRecoveryBlockTime: globalCfg.Qumomf.ShardRecoveryBlockTime,
		InstanceRecoveryBlockTime:   globalCfg.Qumomf.InstanceRecoveryBlockTime,
	}, clusterLogger)
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
