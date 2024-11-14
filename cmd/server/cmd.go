package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "traffikey",
		Short: "A tool to write key/values for traefik configuration",
	}
)

func init() {
	if os.Getenv("DEBUG") != "" {
		log.Info("debug level set")
		log.SetLevel(log.DebugLevel)
	}

	rootCmd.PersistentFlags().StringP("config", "c", "traffikey.json", "json configuration filename")
}

func Execute() error {
	return rootCmd.Execute()
}
