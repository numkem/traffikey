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
}

func NewEtcdManager(cfg *traefikkeymate.Config) (KeymateConnector, error) {
	client, err := etcd.New(etcd.Config{
		Endpoints: cfg.Etcd.Endpoints,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %v", err)
	}

	return &EtcdKeymateManager{
		client: client,
	}, nil
}

func validateTarget(target *traefikkeymate.Target) error {
	// Name cannot be empty
	if target.Name == "" {
		return fmt.Errorf("target name cannot be empty")
	}

	// Prefix cannot be empty
	if target.Prefix == "" {
		return fmt.Errorf("prefix for target named %s cannot be empty", target.Name)
	}

	// Rule cannot be empty
	if target.Rule == "" {
		return fmt.Errorf("prefix for target named %s cannot be empty", target.Name)
	}

	return nil
}

func (m *EtcdKeymateManager) deleteTarget(ctx context.Context, target *traefikkeymate.Target) []error {
	keyPrefixes := []string{
		fmt.Sprintf("%s/http/routers/%s", target.Prefix, target.Name),
		fmt.Sprintf("%s/http/services/%s", target.Prefix, target.Name),
		fmt.Sprintf("%s/http/middlewares/%s", target.Prefix, target.Name),
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
			keys[fmt.Sprintf("%s/http/middlewares/%s/%s/%s", target.Prefix, middleware.Name, middleware.Kind, key)] = value
		}

		middlewareNames = append(middlewareNames, middleware.Name)
	}

	if len(middlewareNames) > 0 {
		keys[fmt.Sprintf("%s/http/routers/%s/middlewares", target.Prefix, target.Name)] = strings.Join(middlewareNames, ",")
	}

	return keys
}

func (m *EtcdKeymateManager) writeTarget(ctx context.Context, target *traefikkeymate.Target) error {
	keys := etcdKeyValue{
		fmt.Sprintf("%s/http/routers/%s/entrypoints", target.Prefix, target.Name): target.Entrypoint,
		fmt.Sprintf("%s/http/routers/%s/rule", target.Prefix, target.Name):        target.Rule,
		fmt.Sprintf("%s/http/routers/%s/service", target.Prefix, target.Name):     target.Name,
	}

	// Set loadbalancing between the endpoints
	for id, url := range target.ServerURLs {
		serverURL := url

		// Check we have a scheme in the url to the server
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			log.Warnf("server URL for target %s doesn't have a scheme, adding http", target.Name)
			serverURL = fmt.Sprintf("http://%s", url)
		}

		keys[fmt.Sprintf("%s/http/services/%s/loadbalancer/servers/%d/url", target.Prefix, target.Name, id)] = serverURL
	}

	if target.TLS {
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

		if err := validateTarget(target); err != nil {
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
