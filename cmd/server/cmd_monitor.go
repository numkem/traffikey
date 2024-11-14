package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	logrusmiddleware "github.com/numkem/echo-logrusmiddleware"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "monitor configured endpoints",
	Long:  "monitor configured endpoints. If one become unreachable, it will redirect to itself and show an error message",
	Run:   monitorCmdRun,
}

const DEFAULT_BIND_ADDRESS = "0.0.0.0:7865"

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.PersistentFlags().StringP("bind", "b", DEFAULT_BIND_ADDRESS, "Binding address for the monitoring server")
}

func monitorCmdRun(cmd *cobra.Command, args []string) {
	configFilename := cmd.Flag("config").Value.String()
	mon, err := NewMonitor(configFilename)
	if err != nil {
		log.Fatalf("failed to read configuration: %v", err)
	}

	h := &handler{monitor: mon}

	// echo init
	e := echo.New()
	e.HideBanner = true
	e.Logger = logrusmiddleware.Logger{Logger: log.StandardLogger()}

	e.GET("/", h.List)

	go func() {
		mon.Start()
		e.Logger.Fatal(e.Start(fmt.Sprintf("%s", cmd.Flag("bind").Value)))
	}()

	// Listen for signal and quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-sig:
		ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Second)

		mon.Stop()
		e.Shutdown(ctx)
		cancel()
	}
}
