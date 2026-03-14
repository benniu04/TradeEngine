package config

import "os"

type Config struct {
	DatabaseURL  string
	RedisAddr    string
	KafkaBrokers []string
	GatewayPort  string
}

func Load() *Config {
	return &Config{
		DatabaseURL:  envOrDefault("DATABASE_URL", "postgres://trade:trade@localhost:5432/tradeengine?sslmode=disable"),
		RedisAddr:    envOrDefault("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: []string{envOrDefault("KAFKA_BROKER", "localhost:9092")},
		GatewayPort:  envOrDefault("GATEWAY_PORT", "8080"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
