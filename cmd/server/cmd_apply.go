package main

import (
	"context"
	"os/signal"
	"syscall"

	traffikey "github.com/numkem/traffikey"
	"github.com/numkem/traffikey/keymate"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var applyConfigCmd = &cobra.Command{
	Use:   "apply",
	Short: "apply the configuration file and write the key/values to the store",
	Run:   applyConfigCmdRun,
}

func init() {
	rootCmd.AddCommand(applyConfigCmd)
	rootCmd.MarkFlagRequired("config")
}

func applyConfigCmdRun(cmd *cobra.Command, args []string) {
	configFilename := cmd.Flag("config").Value.String()

	cfg, err := traffikey.NewConfig(configFilename)
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
