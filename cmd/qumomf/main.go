package main

import (
	"flag"
	"fmt"
<<<<<<< HEAD
=======
	"log"
>>>>>>> e90d7bcae9539f18accec821fc3015b13f3fbf5c
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

<<<<<<< HEAD
	log.Info().Msg("Starting qumomf")

	cluster := vshard.NewCluster(cfg.Routers, cfg.Shards)

=======
	log.Println("Starting qumomf")

	cluster := vshard.NewCluster(cfg.Routers, cfg.Shards)

>>>>>>> e90d7bcae9539f18accec821fc3015b13f3fbf5c
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

<<<<<<< HEAD
	log.Info().Msg(fmt.Sprintf("Received system signal: %s. Shutting down qumomf", sig))
=======
	log.Println(fmt.Sprintf("Received system signal: %s. Shutting down qumomf", sig))
>>>>>>> e90d7bcae9539f18accec821fc3015b13f3fbf5c

	mon.Shutdown()
	failover.Shutdown()
	cluster.Shutdown()
}
