package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

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

type Config struct {
	Server      ServerConfig
	AI          AIConfig
	DatabaseURL string
	JWTSecret   string
	Port        string
}

func Load() (*Config, error) {
	// Переменные контейнера имеют приоритет над локальным .env.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("чтение .env: %w", err)
	}
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "JWT_SECRET"} {
		if os.Getenv(key) == "" {
			return nil, fmt.Errorf("не задана переменная %s", key)
		}
	}
	if len(os.Getenv("JWT_SECRET")) < 32 {
		return nil, fmt.Errorf("JWT_SECRET должен содержать минимум 32 байта")
	}
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil || dbPort < 1 || dbPort > 65535 {
		return nil, fmt.Errorf("некорректный DB_PORT")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("SERVER_PORT")
	}
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return nil, fmt.Errorf("некорректный PORT")
	}

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "0.0.0.0"
	}

	aiBaseURL := os.Getenv("AI_BASE_URL")
	if aiBaseURL == "" {
		aiBaseURL = "http://10.154.0.1:8080/v1"
	}

	aiModel := os.Getenv("AI_MODEL")
	if aiModel == "" {
		aiModel = "gemma4:12b"
	}

	aiTimeout := 30
	if val := os.Getenv("AI_TIMEOUT"); val != "" {
		if t, err := strconv.Atoi(val); err == nil && t > 0 {
			aiTimeout = t
		}
	}

	// URL безопасно экранирует специальные символы в реквизитах подключения.
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:   net.JoinHostPort(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		Path:   "/" + os.Getenv("DB_NAME"),
	}
	q := u.Query()
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	q.Set("sslmode", sslmode)
	q.Set("connect_timeout", "5")
	u.RawQuery = q.Encode()

	return &Config{
		DatabaseURL: u.String(),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        port,
		Server: ServerConfig{
			Addr: serverAddr,
			Port: port,
		},
		AI: AIConfig{
			BaseURL: aiBaseURL,
			Model:   aiModel,
			APIKey:  os.Getenv("AI_API_KEY"),
			Timeout: aiTimeout,
		},
	}, nil
}
