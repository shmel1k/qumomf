package main

import (
	"flag"
	"log"

	"github.com/shmel1k/qumomf/internal/config"
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

	log.Println(cfg)
}
