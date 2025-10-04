package main

import (
	"log"

	"github.com/moehandi/konfig"
)

// Configuration models the settings used by the basic example.
type Configuration struct {
	Server   string
	Port     int
	Debug    bool
	Database struct {
		Type string
		Name string
		Port int
	}
}

func main() {
	var cfg Configuration
	if err := konfig.GetConf("example/basic/config/app", &cfg); err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("server %s:%d (debug=%v)", cfg.Server, cfg.Port, cfg.Debug)
	log.Printf("database %s on %s:%d", cfg.Database.Name, cfg.Database.Type, cfg.Database.Port)
}
