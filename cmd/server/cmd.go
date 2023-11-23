package main

import "github.com/spf13/cobra"

var (
	rootCmd = &cobra.Command{
		Use:   "traffikey",
		Short: "A tool to write key/values for traefik configuration",
	}
)

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "traffikey.json", "json configuration filename")
}

func Execute() error {
	return rootCmd.Execute()
}
