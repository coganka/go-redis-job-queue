package config

import (
	"fmt"
	"os"
)

type Config struct {
	RedisAddr     string
	RedisDB       int
	Stream        string
	ConsumerGroup string
	APIKey        string
	Port          string
}

func Load() Config {
	return Config{
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		Stream:        getEnv("STREAM", "jobs:stream"),
		ConsumerGroup: getEnv("CONSUMER_GROUP", "jobs:cg"),
		APIKey:        getEnv("API_KEY", "devkey"),
		Port:          getEnv("PORT", "8080"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		_, err := fmt.Sscanf(v, "%d", &n)
		if err == nil {
			return n
		}
	}
	return def
}
