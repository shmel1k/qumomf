package main

import (
	"flag"
	"log"
	"time"

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
		log.Fatal(err)
	}

	cluster := vshard.NewCluster(cfg.Shards)

	mon := monitor.New(monitor.Config{
		CheckTimeout: time.Second,
	}, cluster)

	log.Println("Starting qumomf")

	errs := mon.Serve()
	select {
	case <-errs:
	}
}
