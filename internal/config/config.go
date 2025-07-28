package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	NodeID     string   `yaml:"node_id"`
	HTTPAddr   string   `yaml:"http_addr"`
	LeaderAddr string   `yaml:"leader_addr"`
	RaftAddr   string   `yaml:"raft_addr"`
	Peers      []string `yaml:"peers"`
}

// Load reads a YAML file from the given path and returns a Config object
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
