package keymate

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	etcd "go.etcd.io/etcd/client/v3"
	"golang.org/x/exp/maps"

	"github.com/numkem/traffikey"
)

type etcdKeyValue map[string]string

type EtcdKeymateManager struct {
	client *etcd.Client
	cfg    *traffikey.Config
}

const TRAEFIK_DEFAULT_PREFIX = "traefik"

func NewEtcdManager(cfg *traffikey.Config) (KeymateConnector, error) {
	client, err := etcd.New(etcd.Config{
		Endpoints: cfg.Etcd.Endpoints,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %v", err)
	}

	// Validate that the default fields are set in the configuration
	if cfg.Traefik.DefaultEntrypoint == "" {
		return nil, fmt.Errorf("defautl entrypoint cannot be empty")
	}
	if cfg.Traefik.DefaultPrefix == "" {
		log.Warn("applying default traefik prefix")
		cfg.Traefik.DefaultPrefix = TRAEFIK_DEFAULT_PREFIX
	}

	return &EtcdKeymateManager{
		client: client,
		cfg:    cfg,
	}, nil
}

func (m *EtcdKeymateManager) validateTarget(target *traffikey.Target) error {
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

func (m *EtcdKeymateManager) deleteTarget(ctx context.Context, target *traffikey.Target) []error {
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

func valuesForMiddlewares(target *traffikey.Target, middlewares []*traffikey.Middleware) etcdKeyValue {
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

func (m *EtcdKeymateManager) writeTarget(ctx context.Context, target *traffikey.Target) error {
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
		tlsKey := fmt.Sprintf("%s/http/routers/%s/tls", target.Prefix, target.Name)
		keys[tlsKey] = "true"

		for key, value := range target.TLSExtraKeys {
			keys[fmt.Sprintf("%s/%s", tlsKey, key)] = value
		}
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

func (m *EtcdKeymateManager) ApplyConfig(ctx context.Context, cfg *traffikey.Config) []error {
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

func (m *EtcdKeymateManager) middlewaresForRouter(ctx context.Context, routerName string, prefix string) ([]*traffikey.Middleware, error) {
	// Fetch the middlewares names for this router
	resp, err := m.client.KV.Get(ctx, fmt.Sprintf("%s/http/routers/%s/middlewares", prefix, routerName))
	if err != nil {
		return nil, fmt.Errorf("failed to get middlewares key from etcd: %v", err)
	}

	var middlewareNames []string
	if resp != nil && len(resp.Kvs) > 0 {
		middlewareNames = strings.Split(resp.Kvs[0].String(), ",")
	}

	// For each middleware, fetch it's config
	middlewares := make(map[string]traffikey.Middleware)
	for _, middlewareName := range middlewareNames {
		// Get the key/value pair
		mdResp, err := m.client.KV.Get(ctx, fmt.Sprintf("%s/http/middlewares/%s", prefix, middlewareName), etcd.WithPrefix())
		if err != nil {
			return nil, fmt.Errorf("failed to get middleware keys from etcd: %v", err)
		}

		for _, key := range mdResp.Kvs {
			ss := strings.Split(key.String(), "/")

			if _, ok := middlewares[ss[3]]; !ok {
				middlewares[ss[3]] = traffikey.Middleware{
					Name: ss[3],
					Kind: ss[4],
					Values: map[string]string{
						ss[5]: string(key.Value),
					},
				}
			} else {
				middlewares[ss[3]].Values[ss[5]] = string(key.Value)
			}
		}

	}

	var values []*traffikey.Middleware
	for _, v := range middlewares {
		values = append(values, &v)
	}

	return values, nil
}

func (m *EtcdKeymateManager) targetFromRouter(ctx context.Context, routerName string, prefix string) (*traffikey.Target, error) {
	target := traffikey.Target{
		Name:         routerName,
		Type:         "",
		ServerURLs:   []string{},
		Entrypoint:   "",
		Middlewares:  []*traffikey.Middleware{},
		Prefix:       m.cfg.Traefik.DefaultPrefix,
		Rule:         "",
		TLS:          false,
		TLSExtraKeys: map[string]string{},
	}

	serversResp, err := m.client.KV.Get(ctx, fmt.Sprintf("%s/http/services/%s/loadbalancer/servers/", prefix, routerName))
	if err != nil {
		return nil, fmt.Errorf("failed to get loadbalancer keys from etcd: %v", err)
	}
	for _, key := range serversResp.Kvs {
		if strings.Contains(key.String(), "url") {
			target.ServerURLs = append(target.ServerURLs, string(key.Value))
		}
	}

	// Check if the router has TLS
	tlsResp, err := m.client.KV.Get(ctx, fmt.Sprintf("%s/http/routers/%s/tls", prefix, routerName))
	if err != nil {
		return nil, fmt.Errorf("failed to get tls key from etcd: %v", err)
	}
	if tlsResp != nil && len(tlsResp.Kvs) > 0 {
		target.TLS = true
	}

	// Get the router's rules
	ruleResp, err := m.client.KV.Get(ctx, fmt.Sprintf("%s/http/routers/%s/rule", prefix, routerName))
	if err != nil {
		return nil, fmt.Errorf("failed to get rule key from etcd: %v", err)
	}
	if ruleResp != nil && len(ruleResp.Kvs) > 0 {
		target.Rule = string(ruleResp.Kvs[0].Value)
	}

	// Get the router's entrypoint
	entryResp, err := m.client.KV.Get(ctx, fmt.Sprintf("%s/http/routers/%s/entrypoints", prefix, routerName))
	if err != nil {
		return nil, fmt.Errorf("failed to get entrypoint key from etcd: %v", err)
	}
	if entryResp != nil && len(entryResp.Kvs) > 0 {
		target.Entrypoint = string(entryResp.Kvs[0].Value)
	}

	target.Middlewares, err = m.middlewaresForRouter(ctx, routerName, prefix)

	return &target, nil
}

func (m *EtcdKeymateManager) ListTargets(ctx context.Context, cfg *traffikey.Config) ([]*traffikey.Target, error) {
	resp, err := m.client.Get(ctx, cfg.Traefik.DefaultPrefix, etcd.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get targets from etcd: %v", err)
	}

	targets := make(map[string]*traffikey.Target)
	for _, key := range resp.Kvs {
		// Only pick the router keys
		if strings.Contains(key.String(), "router") {
			routerName := strings.Split(key.String(), "/")[3]

			if _, ok := targets[routerName]; !ok {
				t, err := m.targetFromRouter(ctx, routerName, cfg.Traefik.DefaultPrefix)
				if err != nil {
					return nil, fmt.Errorf("failed to get target for router %s from etcd: %v", routerName, err)
				}

				targets[routerName] = t
			}
		}
	}

	var values []*traffikey.Target
	for _, v := range targets {
		values = append(values, v)
	}

	return values, nil
}
