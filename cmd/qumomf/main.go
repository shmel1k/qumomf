package main

import (
	"context"
	"flag"
	"log/syslog"
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
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
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

	logger.Info().Msgf("Starting qumomf %s, commit %s, built at %s", version, commit, buildDate)

	go func() {
		logger.Info().Msgf("Listening on %s", cfg.Qumomf.Port)

		err = server.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to listen HTTP server")
		}
	}()

	if len(cfg.Clusters) == 0 {
		logger.Warn().Msg("No clusters are found in the configuration")
	}

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

	base := zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()

	if cfg.Qumomf.EnableSysLog {
		w, err := syslog.New(syslog.LOG_INFO, "qumomf")
		if err != nil {
			log.Warn().Err(err).Msg("Unable to connect to the system log daemon")
			return base
		}
		syslogWriter := zerolog.SyslogLevelWriter(w)

		return zerolog.New(zerolog.MultiLevelWriter(os.Stdout, syslogWriter)).Level(logLevel).With().Timestamp().Logger()
	}

	return base
}

func initHTTPServer(port string) *http.Server {
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Init routing.
	http.Handle("/debug/metrics", promhttp.Handler())
	http.Handle("/debug/health", qumhttp.HealthHandler())
	http.Handle("/debug/about", qumhttp.AboutHandler(version, commit, buildDate))

	return server
}
