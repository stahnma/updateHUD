package main

import "os"

type Config struct {
	NATSURL  string
	DBPath   string
	HTTPPort string
}

func LoadConfig() Config {
	return Config{
		NATSURL:  getEnv("NATS_URL", "nats://localhost:4222"),
		DBPath:   getEnv("DB_PATH", "systems.db"),
		HTTPPort: getEnv("HTTP_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
