package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/vshard"
	"github.com/shmel1k/qumomf/pkg/vshard/orchestrator"
)

var (
	configPath = flag.String("config", "", "Config file path")
)

func main() {
	flag.Parse()
	cfg, err := config.Setup(*configPath)
	if err != nil {
		log.Error().Msgf("Error happened while setup config: %s", err.Error())
		return
	}

	log.Info().Msg("Starting qumomf")

	cluster := vshard.NewCluster(cfg.Routers, cfg.Shards)

	mon := orchestrator.NewMonitor(orchestrator.Config{
		CheckTimeout: time.Second, // TODO: move to config
	}, cluster)

	elector := quorum.NewLagQuorum()
	failover := orchestrator.NewSwapMasterFailover(cluster, elector)

	analysisStream := mon.Serve()
	failover.Serve(analysisStream)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs

	log.Info().Msgf("Received system signal: %s. Shutting down qumomf", sig)

	mon.Shutdown()
	failover.Shutdown()
	cluster.Shutdown()
}
