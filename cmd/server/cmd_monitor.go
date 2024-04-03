package main

import (
	"fmt"

	"github.com/labstack/echo/v4"
	logrusmiddleware "github.com/numkem/echo-logrusmiddleware"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "monitor configured endpoints",
	Long:  "monitor configured endpoints. If one become unreachable, it will redirect to itself and show an error message explaining the target is not available.",
	Run:   monitorCmdRun,
}

const DEFAULT_BIND_ADDRESS = "0.0.0.0:7865"

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.PersistentFlags().StringP("bind", "b", DEFAULT_BIND_ADDRESS, "Binding address for the monitoring server")
}

func monitorCmdRun(cmd *cobra.Command, args []string) {
	e := echo.New()
	e.Logger = logrusmiddleware.Logger{Logger: log.StandardLogger()}

	e.GET("/", func(c echo.Context) error {
		return nil
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf("%s", cmd.Flag("bind").Value)))
}
