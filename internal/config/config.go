package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr         string
	NATSURL          string
	MonitorEndpoints []string
	ConnectionStore  string
	SecretKeyFile    string
}

func Load() Config {
	return Config{
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		NATSURL:          getEnv("NATS_URL", "nats://127.0.0.1:4222"),
		MonitorEndpoints: splitCSV(getEnv("NATS_MONITOR_URLS", "http://127.0.0.1:8222")),
		ConnectionStore:  getEnv("NATS_CONNECTION_STORE", "data/connections.json"),
		SecretKeyFile:    getEnv("NATS_SECRET_KEY_FILE", "data/secret.key"),
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, strings.TrimRight(part, "/"))
	}
	return items
}
