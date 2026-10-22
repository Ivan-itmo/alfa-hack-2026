package config

import (
	"os"
)

type Config struct {
	Server ServerConfig
	AI     AIConfig
}

type ServerConfig struct {
	Addr string
	Port string
}

type AIConfig struct {
	BaseURL string
	Model   string
	APIKey  string
	Timeout int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Addr: getEnv("SERVER_ADDR", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8081"),
		},
		AI: AIConfig{
			BaseURL: getEnv("AI_BASE_URL", "http://10.154.0.1:8080/v1"),
			Model:   getEnv("AI_MODEL", "gemma4:12b"),
			APIKey:  getEnv("AI_API_KEY", ""),
			Timeout: getIntEnv("AI_TIMEOUT", 30),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if value != "" {
			// Simplified for brevity
		}
	}
	return defaultValue
}
