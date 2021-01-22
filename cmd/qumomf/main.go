package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/shmel1k/qumomf/internal/api"

	"github.com/gorilla/mux"

	"github.com/shmel1k/qumomf/internal/storage"
	"github.com/shmel1k/qumomf/internal/storage/sqlite"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
	"gopkg.in/natefinch/lumberjack.v2"

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

	db, err := newStorage(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init persistent storage")
	}

	service := api.NewService(db)
	server := initHTTPServer(logger, service, cfg.Qumomf.Port)

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

	qCoordinator := coordinator.New(logger, db)
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

func newStorage(cfg *config.Config) (storage.Storage, error) {
	return sqlite.New(sqlite.Config{
		FileName:       cfg.Qumomf.Storage.Filename,
		ConnectTimeout: cfg.Qumomf.Storage.ConnectTimeout,
		QueryTimeout:   cfg.Qumomf.Storage.QueryTimeout,
	})
}

func initLogger(cfg *config.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	loggingCfg := cfg.Qumomf.Logging

	logLevel, err := zerolog.ParseLevel(loggingCfg.Level)
	if err != nil {
		log.Warn().Msgf("Unknown Level String: '%s', defaulting to DebugLevel", loggingCfg.Level)
		logLevel = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(logLevel)

	writers := make([]io.Writer, 0, 1)
	writers = append(writers, os.Stdout)

	if loggingCfg.SysLogEnabled {
		w, err := syslog.New(syslog.LOG_INFO, "qumomf")
		if err != nil {
			log.Warn().Err(err).Msg("Unable to connect to the system log daemon")
		} else {
			writers = append(writers, zerolog.SyslogLevelWriter(w))
		}
	}

	if loggingCfg.FileLoggingEnabled {
		w, err := newRollingLogFile(&loggingCfg)
		if err != nil {
			log.Warn().Err(err).Msg("Unable to init file logger")
		} else {
			writers = append(writers, w)
		}
	}

	var baseLogger zerolog.Logger
	if len(writers) == 1 {
		baseLogger = zerolog.New(writers[0])
	} else {
		return zerolog.New(zerolog.MultiLevelWriter(writers...))
	}

	return baseLogger.Level(logLevel).With().Timestamp().Logger()
}

func newRollingLogFile(cfg *config.Logging) (io.Writer, error) {
	dir := path.Dir(cfg.Filename)
	if unix.Access(dir, unix.W_OK) != nil {
		return nil, fmt.Errorf("no permissions to write logs to dir: %s", dir)
	}

	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxBackups: cfg.MaxBackups,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
	}, nil
}

func initHTTPServer(logger zerolog.Logger, service api.Service, port string) *http.Server {
	r := mux.NewRouter()
	qumhttp.RegisterDebugHandlers(r, version, commit, buildDate)
	qumhttp.RegisterAPIHandlers(r, qumhttp.NewHandler(logger, service))

	return &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}
