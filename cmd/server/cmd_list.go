package main

import (
	"os"

	"github.com/numkem/traffikey"
	"github.com/numkem/traffikey/keymate"

	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "lists all targets for a prefix",
	Run:   listCmdRun,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.PersistentFlags().StringP("prefix", "p", keymate.TRAEFIK_DEFAULT_PREFIX, "etcd key prefix")
}

// Take the argument from the command and look through matching keys in etcd
func listCmdRun(cmd *cobra.Command, args []string) {
	configFilename := cmd.Flag("config").Value.String()
	prefix := cmd.Flag("prefix").Value.String()

	cfg, err := traffikey.NewConfig(configFilename)
	if err != nil {
		log.Fatalf("Failed to read configuraiton: %v", err)
	}

	// Create manager connection
	mgr, err := keymate.NewEtcdManager(cfg)
	if err != nil {
		log.Fatalf("failed to create manager: %v", err)
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Entrypoint", "Middleware", "Prefix", "Rule", "TLS"})

	cfg.Traefik.DefaultPrefix = prefix

	targets, err := mgr.ListTargets(cmd.Context(), cfg)
	for _, target := range targets {
		log.Debugf("Processing target %+v\n", target)

		t.AppendRow(table.Row{target.Name, target.Entrypoint, len(target.Middlewares), target.Prefix, target.Rule, target.TLS})
	}

	t.Render()
}
