package exporter

import (
	"fmt"

	"git.digineo.de/digineo/unifi-sdn-exporter/unifi"
	"github.com/BurntSushi/toml"
)

type Config struct {
	// list of Unifi SDN controllers
	Controllers []*unifi.Controller `toml:"unifi-controller"`

	// Transformed controller instances. Key it the clients target identifier,
	// i.e. the controller alias or the URL's host name.
	clients map[string]unifi.Client
}

// LoadConfig loads the configuration from a file.
func LoadConfig(file string) (*Config, error) {
	cfg := Config{}
	if _, err := toml.DecodeFile(file, &cfg); err != nil {
		return nil, fmt.Errorf("loading config file %q failed: %w", file, err)
	}

	cfg.clients = make(map[string]unifi.Client)
	for i, ctrl := range cfg.Controllers {
		client, err := unifi.NewClient(ctrl)
		if err != nil {
			return nil, fmt.Errorf("invalid controller #%d (%v): %w", i, ctrl, err)
		}
		cfg.clients[client.TargetName()] = client
	}

	return &cfg, nil
}
