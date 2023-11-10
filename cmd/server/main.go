package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	traefikkeymate "github.com/numkem/traefik-keymate"
	"github.com/numkem/traefik-keymate/keymate"
	log "github.com/sirupsen/logrus"
)

func main() {
	configFilename := flag.String("config", "", "configuration filename")
	flag.Parse()

	if *configFilename == "" {
		log.Fatal("configuration filename required")
	}

	log.Info("Traefik Keymate starting...")

	cfg, err := traefikkeymate.NewConfig(*configFilename)
	if err != nil {
		log.Fatalf("Failed to read configuraiton: %v", err)
	}

	// Check if defaults are set
	if cfg.Traefik.DefaultPrefix == "" {
		log.Warn("default traefik prefix isn't set, this could cause errors")
	}

	if cfg.Traefik.DefaultEntrypoint == "" {
		log.Warn("default Traefik entrypoint isn't set, this could cause errors")
	}

	// Create manager connection
	mgr, err := keymate.NewEtcdManager(cfg)
	if err != nil {
		log.Fatalf("failed to create manager: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	errs := mgr.ApplyConfig(ctx, cfg)
	for _, err := range errs {
		log.Errorf("error found while applying configuration: %v", err)
	}

	for _, err := range errs {
		log.Error(err)
	}

	log.Info("configuration applied. Exiting")
}
