package main

import (
	"log"
	"os"

	"github.com/moehandi/konfig"
)

// Credentials holds the environment-driven credentials for the env example.
type Credentials struct {
	Username string `env:"APP_USER"`
	Password string `env:"APP_PASS"`
}

// Config bundles the settings loaded from environment variables and fallback files.
type Config struct {
	Address      string
	Port         int
	Credentials  Credentials
	TimeoutSecs  *int `json:"timeout_secs"`
	FeatureFlags struct {
		Beta bool
	}
}

func main() {
	// Simulate external environment
	os.Setenv("APP_ADDRESS", "env.example.local")
	os.Setenv("APP_PORT", "8443")
	os.Setenv("APP_TIMEOUT_SECS", "45")
	os.Setenv("APP_FEATURE_FLAGS_BETA", "true")
	os.Setenv("APP_USER", "service")
	os.Setenv("APP_PASS", "super-secret")

	var cfg Config
	if err := konfig.Load(&cfg, konfig.WithEnvPrefix("APP"), konfig.WithFiles("example/env/config/fallback.json")); err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("address: %s:%d", cfg.Address, cfg.Port)
	if cfg.TimeoutSecs != nil {
		log.Printf("timeout: %d", *cfg.TimeoutSecs)
	}
	log.Printf("feature flags: %+v", cfg.FeatureFlags)
	log.Printf("credentials: %s/%s", cfg.Credentials.Username, cfg.Credentials.Password)
}
