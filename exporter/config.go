package exporter

import (
	"fmt"

	"git.digineo.de/digineo/unifi-sdn-exporter/unifi"
	"github.com/BurntSushi/toml"
)

type Config struct {
	// list of Unifi SDN controllers
	Controllers []*unifi.Controller `toml:"unifi-controller"`
}

// LoadConfig loads the configuration from a file.
func LoadConfig(file string) (*Config, error) {
	cfg := Config{}

	if _, err := toml.DecodeFile(file, &cfg); err != nil {
		return nil, fmt.Errorf("loading config file %q failed: %w", file, err)
	}

	return &cfg, nil
}

// getClient builds a client.
func (cfg *Config) getClient(target string) (unifi.Client, error) {
	for _, ctrl := range cfg.Controllers {
		if target == ctrl.Alias || target == ctrl.URL {
			return unifi.NewClient(ctrl)
		}
	}

	return nil, nil
}
