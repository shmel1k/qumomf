package main

import (
	"flag"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/pkg/vshard"
	"github.com/shmel1k/qumomf/pkg/vshard/monitor"
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

	cluster := vshard.NewCluster(cfg.Shards)

	mon := monitor.New(monitor.Config{
		CheckTimeout: time.Second,
	}, cluster)

	log.Info().Msg("Starting qumomf")

	errs := mon.Serve()
	select {
	case <-errs:
	}
}
