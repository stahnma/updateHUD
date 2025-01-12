package main

import (
	"log"
	"os"
)

type Config struct {
	NATSURL  string
	DBPath   string
	HTTPPort string
}

func LoadConfig() Config {
	config := Config{
		NATSURL:  getEnv("NATS_URL", "nats://admin:password@localhost:4222"),
		DBPath:   getEnv("DB_PATH", "systems.db"),
		HTTPPort: getEnv("HTTP_PORT", "8080"),
	}
	log.Printf("Loaded configuration: %+v", config)
	return config
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
