package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr       string
	GRPCAddr       string
	RKNAPIBaseURL  string
	UpdateInterval time.Duration
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:      getenv("HTTP_ADDR", ":80"),
		GRPCAddr:      getenv("GRPC_ADDR", ":9090"),
		RKNAPIBaseURL: getenv("RKN_API_BASE_URL", "https://reestr.rublacklist.net/api/v3"),
	}

	intervalStr := getenv("UPDATE_INTERVAL", "6h")
	d, err := time.ParseDuration(intervalStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid UPDATE_INTERVAL=%q: %w", intervalStr, err)
	}
	if d < time.Hour {
		return Config{}, fmt.Errorf("UPDATE_INTERVAL too small (%s), must be >=1h", d)
	}
	if d > 48*time.Hour {
		return Config{}, fmt.Errorf("UPDATE_INTERVAL too large (%s), must be <=48h", d)
	}
	cfg.UpdateInterval = d

	if cfg.RKNAPIBaseURL == "" {
		return Config{}, fmt.Errorf("RKN_API_BASE_URL must not be empty")
	}

	return cfg, nil
}
