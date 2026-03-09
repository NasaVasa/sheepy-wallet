package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Config ServerConfig `json:"config"`
	Gates  []Gate       `json:"gates"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Gate holds wallet gate configuration. The mnemonic is sensitive — never log it.
type Gate struct {
	Name     string `json:"name"`
	Mnemonic string `json:"mnemonic"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return &cfg, nil
}

func (c *Config) FindGate(name string) (*Gate, error) {
	for i := range c.Gates {
		if c.Gates[i].Name == name {
			return &c.Gates[i], nil
		}
	}
	return nil, fmt.Errorf("gate %q not found", name)
}
