package traefikkeymate

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Targets []*Target      `json:"targets"`
	Etcd    *etcdConfig    `json:"etcd"`
	Traefik *traefikConfig `json:"traefik"`
}

type etcdConfig struct {
	Endpoints []string `json:"endpoints"`
	SSL       bool     `json:"ssl"`
}

type traefikConfig struct {
	DefaultPrefix     string `json:"default_prefix"`
	DefaultEntrypoint string `json:"default_entrypoint"`
}

func NewConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file %s doesn't exists", filename)
		}

		return nil, fmt.Errorf("failed to read configuration file %s: %v", filename, err)
	}

	cfg := new(Config)
	err = json.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return cfg, nil
}
