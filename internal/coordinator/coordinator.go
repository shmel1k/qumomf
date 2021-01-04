package coordinator

import (
	"errors"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/internal/quorum"
	"github.com/shmel1k/qumomf/internal/storage"
	"github.com/shmel1k/qumomf/internal/vshard"
	"github.com/shmel1k/qumomf/internal/vshard/orchestrator"

	"github.com/rs/zerolog"
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

func (c *Coordinator) RegisterCluster(name string, relStorage storage.Storage, cfg config.ClusterConfig, globalCfg *config.Config) error {
	if _, exist := c.clusters[name]; exist {
		return ErrClusterAlreadyExist
	}

	clusterLogger := c.logger.With().Str("cluster", name).Logger()

	cluster := vshard.NewCluster(name, relStorage, cfg)
	cluster.SetLogger(clusterLogger)
	c.clusters[name] = cluster
	c.addShutdownTask(cluster.Shutdown)

	mon := orchestrator.NewMonitor(cluster, orchestrator.Config{
		RecoveryPollTime:  globalCfg.Qumomf.ClusterRecoveryTime,
		DiscoveryPollTime: globalCfg.Qumomf.ClusterDiscoveryTime,
	}, clusterLogger)
	c.addShutdownTask(mon.Shutdown)

	hooker := initHooker(globalCfg, clusterLogger)
	elector := quorum.New(quorum.Mode(*cfg.ElectionMode), quorum.Options{
		ReasonableFollowerLSNLag: globalCfg.Qumomf.ReasonableFollowerLSNLag,
		ReasonableFollowerIdle:   globalCfg.Qumomf.ReasonableFollowerIdle.Seconds(),
	})
	failover := orchestrator.NewDefaultFailover(cluster, relStorage, orchestrator.FailoverConfig{
		Hooker:                      hooker,
		Elector:                     elector,
		ReplicaSetRecoveryBlockTime: globalCfg.Qumomf.ShardRecoveryBlockTime,
		InstanceRecoveryBlockTime:   globalCfg.Qumomf.InstanceRecoveryBlockTime,
	}, clusterLogger)
	c.addShutdownTask(failover.Shutdown)

	analysisStream := mon.Serve()
	failover.Serve(analysisStream)

	return nil
}

func (c *Coordinator) Shutdown() {
	for i := len(c.shutdownQueue) - 1; i >= 0; i-- {
		task := c.shutdownQueue[i]
		task()
	}
}

func (c *Coordinator) addShutdownTask(task shutdownTask) {
	c.shutdownQueue = append(c.shutdownQueue, task)
}

func initHooker(cfg *config.Config, logger zerolog.Logger) *orchestrator.Hooker {
	hooksCfg := cfg.Qumomf.Hooks
	hooker := orchestrator.NewHooker(hooksCfg.Shell, logger)
	hooker.SetTimeout(hooksCfg.Timeout)
	hooker.SetTimeoutAsync(hooksCfg.TimeoutAsync)

	hooker.AddHook(orchestrator.HookPreFailover, hooksCfg.PreFailover...)
	hooker.AddHook(orchestrator.HookPostSuccessfulFailover, hooksCfg.PostSuccessfulFailover...)
	hooker.AddHook(orchestrator.HookPostUnsuccessfulFailover, hooksCfg.PostUnsuccessfulFailover...)

	return hooker
}
