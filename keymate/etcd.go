package keymate

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	etcd "go.etcd.io/etcd/client/v3"
	"golang.org/x/exp/maps"

	traefikkeymate "github.com/numkem/traefik-keymate"
)

type etcdKeyValue map[string]string

type EtcdKeymateManager struct {
	client *etcd.Client
	cfg    *traefikkeymate.Config
}

func NewEtcdManager(cfg *traefikkeymate.Config) (KeymateConnector, error) {
	client, err := etcd.New(etcd.Config{
		Endpoints: cfg.Etcd.Endpoints,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %v", err)
	}

	// Validate that the default fields are set in the configuraiton
	if cfg.Traefik.DefaultEntrypoint == "" {
		return nil, fmt.Errorf("defautl entrypoint cannot be empty")
	}
	if cfg.Traefik.DefaultPrefix == "" {
		return nil, fmt.Errorf("default prefix cannot be empty")
	}

	return &EtcdKeymateManager{
		client: client,
		cfg:    cfg,
	}, nil
}

func (m *EtcdKeymateManager) validateTarget(target *traefikkeymate.Target) error {
	// Name cannot be empty
	if target.Name == "" {
		return fmt.Errorf("target name cannot be empty")
	}

	// Prefix cannot be empty
	if target.Prefix == "" {
		target.Prefix = m.cfg.Traefik.DefaultPrefix
	}

	if target.Entrypoint == "" {
		target.Entrypoint = m.cfg.Traefik.DefaultEntrypoint
	}

	// Rule cannot be empty
	if target.Rule == "" {
		return fmt.Errorf("prefix for target named %s cannot be empty", target.Name)
	}

	if target.Type == "" {
		log.Warnf("target %s has an empty type, using http...", target.Name)
		target.Type = "http"
	}

	return nil
}

func (m *EtcdKeymateManager) deleteTarget(ctx context.Context, target *traefikkeymate.Target) []error {
	err := m.validateTarget(target)
	if err != nil {
		return []error{fmt.Errorf("invalid target: %v", err)}
	}

	keyPrefixes := []string{
		fmt.Sprintf("%s/%s/routers/%s", target.Prefix, target.Type, target.Name),
		fmt.Sprintf("%s/%s/services/%s", target.Prefix, target.Type, target.Name),
		fmt.Sprintf("%s/%s/middlewares/%s", target.Prefix, target.Type, target.Name),
	}

	var errs []error
	for _, key := range keyPrefixes {
		_, err := m.client.KV.Delete(ctx, key, etcd.WithPrefix())
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete key prefix for target %s: %v", target.Name, err))
		}
	}

	return errs
}

func valuesForMiddlewares(target *traefikkeymate.Target, middlewares []*traefikkeymate.Middleware) etcdKeyValue {
	keys := make(etcdKeyValue)
	var middlewareNames []string
	for _, middleware := range middlewares {
		for key, value := range middleware.Values {
			keys[fmt.Sprintf("%s/%s/middlewares/%s/%s/%s", target.Prefix, target.Type, middleware.Name, middleware.Kind, key)] = value
		}

		middlewareNames = append(middlewareNames, middleware.Name)
	}

	if len(middlewareNames) > 0 {
		keys[fmt.Sprintf("%s/%s/routers/%s/middlewares", target.Prefix, target.Type, target.Name)] = strings.Join(middlewareNames, ",")
	}

	return keys
}

func (m *EtcdKeymateManager) writeTarget(ctx context.Context, target *traefikkeymate.Target) error {
	keys := etcdKeyValue{
		fmt.Sprintf("%s/%s/routers/%s/entrypoints", target.Prefix, target.Type, target.Name): target.Entrypoint,
		fmt.Sprintf("%s/%s/routers/%s/rule", target.Prefix, target.Type, target.Name):        target.Rule,
		fmt.Sprintf("%s/%s/routers/%s/service", target.Prefix, target.Type, target.Name):     target.Name,
	}

	// Set loadbalancing between the endpoints
	for id, url := range target.ServerURLs {
		serverURL := url

		// Check we have a scheme in the url to the server with http routers
		if target.Type == "http" {
			if !strings.Contains(url, "//") {
				log.Warnf("server URL for target %s doesn't have a scheme, adding %s", target.Type, target.Name)
				serverURL = fmt.Sprintf("%s://%s", target.Type, url)
			}
		}

		suffix := "url"
		if target.Type != "http" {
			suffix = "address"
		}

		keys[fmt.Sprintf("%s/%s/services/%s/loadbalancer/servers/%d/%s", target.Prefix, target.Type, target.Name, id, suffix)] = serverURL
	}

	if target.TLS && target.Type == "http" {
		keys[fmt.Sprintf("%s/http/routers/%s/tls", target.Prefix, target.Name)] = "true"
	}

	// Apply all the middlewares
	maps.Copy(keys, valuesForMiddlewares(target, target.Middlewares))

	// Write the key/value
	for key, value := range keys {
		_, err := m.client.KV.Put(ctx, key, value)
		if err != nil {
			if e := m.deleteTarget(ctx, target); e != nil {
				log.Warnf("failed to cleanup the target after failed insertion: %v", e)
			}

			return fmt.Errorf("failed to write key in etcd: %v", err)
		}
	}

	return nil
}

func (m *EtcdKeymateManager) ApplyConfig(ctx context.Context, cfg *traefikkeymate.Config) []error {
	var errs []error

	for _, target := range cfg.Targets {
		errs = m.deleteTarget(ctx, target)
		if len(errs) != 0 {
			return errs
		}

		// Validate the target's configuration
		if target.Prefix == "" {
			target.Prefix = cfg.Traefik.DefaultPrefix
		}
		if target.Entrypoint == "" {
			target.Entrypoint = cfg.Traefik.DefaultEntrypoint
		}

		if err := m.validateTarget(target); err != nil {
			errs = append(errs, err)
			continue
		}

		err := m.writeTarget(ctx, target)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
