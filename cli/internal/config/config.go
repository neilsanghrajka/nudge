// Package config manages stake configuration.
package config

import (
	"github.com/neilsanghrajka/stake-ai/cli/internal/store"
)

type Config struct {
	Punishments       map[string]map[string]any `json:"punishments"`
	DefaultPunishment string                    `json:"default_punishment"`
}

func Load() *Config {
	c := &Config{
		Punishments: map[string]map[string]any{},
	}
	store.LoadJSON("config.yaml.json", c)
	if c.Punishments == nil {
		c.Punishments = map[string]map[string]any{}
	}
	return c
}

func Save(c *Config) error {
	return store.SaveJSON("config.yaml.json", c)
}
