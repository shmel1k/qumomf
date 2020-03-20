package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/internal/coordinator"
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

	qCoordinator := coordinator.New()
	for clusterName, clusterCfg := range cfg.Clusters {
		err = qCoordinator.RegisterCluster(clusterName, clusterCfg)
		if err != nil {
			log.Error().Err(err).Msgf("Could not register cluster with name %s", clusterName)
			continue
		}
		log.Info().Msgf("New cluster '%s' has been registered", clusterName)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs

	log.Info().Msgf("Received system signal: %s. Shutting down qumomf", sig)
	qCoordinator.Shutdown()
}
