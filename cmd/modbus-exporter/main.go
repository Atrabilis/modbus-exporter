package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/atrabilis/modbus-exporter/internal/config"
	"github.com/atrabilis/modbus-exporter/internal/httpserver"
	"github.com/atrabilis/modbus-exporter/internal/modbus"
	"github.com/atrabilis/modbus-exporter/internal/store"
)

func main() {
	configPath := flag.String(
		"config",
		"config/example.yml",
		"Path to configuration file",
	)
	listenAddr := flag.String(
		"listen",
		":9105",
		"HTTP listen address",
	)
	debug := flag.Bool(
		"debug",
		false,
		"Enable debug logging",
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	if *debug {
		log.Printf("debug enabled")
	}

	st := store.New()
	poller := modbus.NewPoller(cfg, st, *debug)
	server := httpserver.New(*listenAddr, st)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go poller.Run(ctx)

	go func() {
		if err := server.Run(); err != nil {
			log.Fatalf("http server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
	log.Printf("shutting down")
}
