package main

import (
	"log"

	"github.com/moehandi/konfig"
)

// AppConfig demonstrates layered configuration overrides in the multi example.
type AppConfig struct {
	Server string
	Port   int
	Debug  bool
	Labels map[string]string `json:"labels"`
}

func main() {
	var cfg AppConfig

	err := konfig.Load(&cfg,
		konfig.WithFiles(
			"example/multi/config/default.yaml",
			"example/multi/config/override.toml",
		),
	)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("server %s:%d (debug=%v)", cfg.Server, cfg.Port, cfg.Debug)
	log.Printf("labels: %#v", cfg.Labels)
}
