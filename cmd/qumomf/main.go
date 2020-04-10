package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
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
		log.Fatal().Err(err).Msgf("failed to read config")
	}

	logger := initLogger(cfg)

	logger.Info().Msg("Starting qumomf")

	qCoordinator := coordinator.New(logger)
	for clusterName, clusterCfg := range cfg.Clusters {
		err = qCoordinator.RegisterCluster(clusterName, clusterCfg, cfg)
		if err != nil {
			logger.Err(err).Msgf("Could not register cluster with name %s", clusterName)
			continue
		}
		logger.Info().Msgf("New cluster '%s' has been registered", clusterName)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	sig := <-interrupt

	logger.Info().Msgf("Received system signal: %s. Shutting down qumomf", sig)
	qCoordinator.Shutdown()
}

func initLogger(cfg *config.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	logLevel, err := zerolog.ParseLevel(cfg.Qumomf.LogLevel)
	if err != nil {
		log.Warn().Msgf("Unknown Level String: '%s', defaulting to DebugLevel", cfg.Qumomf.LogLevel)
		logLevel = zerolog.DebugLevel
	}

	return zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()
}
