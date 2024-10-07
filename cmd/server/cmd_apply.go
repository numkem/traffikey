package main

import (
	"os/signal"
	"syscall"

	traffikey "github.com/numkem/traffikey"
	"github.com/numkem/traffikey/keymate"

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
		cmd.PrintErrf("ERR: failed to read configuraiton: %v", err)
		return
	}

	// Check if defaults are set
	if cfg.Traefik.DefaultPrefix == "" {
		cmd.Printf("WARN: default traefik prefix isn't set, this could cause errors\n")
	}

	if cfg.Traefik.DefaultEntrypoint == "" {
		cmd.Printf("WARN: default Traefik entrypoint isn't set, this could cause errors\n")
	}

	// Create manager connection
	mgr, err := keymate.NewEtcdManager(cfg)
	if err != nil {
		cmd.PrintErrf("ERR: failed to create manager: %v\n", err)
		return
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	errs := mgr.ApplyConfig(ctx, cfg)
	for _, err := range errs {
		cmd.PrintErrf("ERR: error found while applying configuration: %v\n", err)
		return
	}

	// Get previous state and compare to see if some tagerts were removed
	oldState, err := mgr.GetState(ctx)
	if err != nil {
		cmd.PrintErrf("ERR: failed to get previous state: %v\n", err)
		return
	}

	if oldState != nil {
		// Go through the old state and check for each target to see if they still exists
		for _, ot := range oldState.Targets {
			var found bool
			for _, t := range cfg.Targets {
				if ot.Name == t.Name {
					found = true
				}
			}

			if !found {
				prefix := oldState.Traefik.DefaultPrefix
				if ot.Prefix != "" {
					prefix = ot.Prefix
				}

				cmd.Printf("INF: deleting removed target %s\n", ot.Name)
				err = mgr.DeleteTargetByName(ctx, ot.Name, prefix)
				if err != nil {
					cmd.PrintErrf("ERR: failed to delete old target named %s: %v", ot.Name, err)
					return
				}
			}
		}
	}

	err = mgr.SaveState(ctx, cfg)
	if err != nil {
		cmd.PrintErrf("ERR: failed to write state: %v", err)
		return
	}

	cmd.Print("configuration applied!\n")
}
