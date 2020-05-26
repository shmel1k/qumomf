package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/internal/coordinator"
	"github.com/shmel1k/qumomf/internal/qumhttp"
)

var (
	Version   string
	BuildDate string
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
	server := initHTTPServer(cfg.Qumomf.Port)

	logger.Info().Msg("Starting qumomf")

	go func() {
		logger.Info().Msgf("Listening on %s", cfg.Qumomf.Port)

		err = server.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Err(err).Msg("Failed to listen HTTP server")
		}
	}()

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

	err = server.Shutdown(context.Background())
	if err != nil {
		logger.Err(err).Msg("Failed to shutting down the HTTP server gracefully")
	}
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

func initHTTPServer(port string) *http.Server {
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Init routing.
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/health", qumhttp.HealthHandler())
	http.Handle("/about", qumhttp.AboutHandler(Version, BuildDate))

	return server
}
