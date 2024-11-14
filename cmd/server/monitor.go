package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	traffikey "github.com/numkem/traffikey"
	"github.com/numkem/traffikey/keymate"

	log "github.com/sirupsen/logrus"
)

type Monitor struct {
	currentTargets *sync.Map
	manager        keymate.KeymateConnector
	cfg            *traffikey.Config
}

func NewMonitor(configFilename string) (*Monitor, error) {
	// Read configuration file
	cfg, err := traffikey.NewConfig(configFilename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read configuraiton: %v", err)
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

	return &Monitor{cfg: cfg, currentTargets: &sync.Map{}, manager: mgr}, nil
}

func testTarget(tgt *traffikey.Target) []string {
	// Attempt a test connection to the target
	switch tgt.Type {
	case "http":
		return testHTTPTarget(tgt)
	default:
		log.WithField("target", tgt.Name).Warnf("invalid target type %s", tgt.Type)
	}

	return []string{}
}

func testHTTPTarget(tgt *traffikey.Target) []string {
	// Try to connect to the http endpoint with a very small timeout
	aliveUrls := []string{}
	for _, url := range tgt.ServerURLs {
		client := http.Client{
			Timeout: 1 * time.Second,
		}

		_, err := client.Get(url)
		// TODO: Check the HTTP response code
		if err == nil {
			aliveUrls = append(aliveUrls, url)
		}
	}

	return aliveUrls
}

type monitoredTarget struct {
	ID      string
	Context context.Context
	Cancel  context.CancelFunc
	Target  *traffikey.Target
}

func (m *Monitor) Start() {
	// Start a monitor goroutine for each target
	for _, tgt := range m.cfg.Targets {
		if !tgt.Monitored {
			log.WithField("target", tgt.Name).Infof("target %s isn't monitored", tgt.Name)
			continue
		}

		id, err := uuid.NewV4()
		if err != nil {
			log.WithField("target", tgt.Name).Errorf("failed to generate UUID: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		mt := &monitoredTarget{
			ID:      id.String(),
			Context: ctx,
			Cancel:  cancel,
			Target:  tgt,
		}
		m.currentTargets.Store(id, mt)

		log.WithField("target", tgt.Name).Debug("starting monitoring of target")
		go watchTarget(mt)
	}
}

func (m *Monitor) Stop() {
	m.currentTargets.Range(func(key, value interface{}) bool {
		mt := value.(*monitoredTarget)

		log.Debugf("Stopping monitoring of target ID %s :: %s", mt.ID, mt.Target.Name)
		mt.Cancel()
		return true
	})
}

func watchTarget(m *monitoredTarget) {
	for {
		ticker := time.NewTicker(15 * time.Second)

		aliveUrls := testTarget(m.Target)
		switch len(aliveUrls) {
		case 0:
			log.WithField("target", m.Target.Name).Infof("Target is DOWN (0/%d)", len(m.Target.ServerURLs))
		default:
			upNb := len(m.Target.ServerURLs) - len(aliveUrls)
			log.WithField("target", m.Target.Name).Infof("Target is UP (%d/%d)", upNb, len(m.Target.ServerURLs))
		}

		select {
		case <-m.Context.Done():
			return

		case <-ticker.C:
		}
	}
}
