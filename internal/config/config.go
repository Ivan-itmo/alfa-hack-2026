package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
}

func Load() (Config, error) {
	// Переменные контейнера имеют приоритет над локальным .env.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("чтение .env: %w", err)
	}
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "JWT_SECRET"} {
		if os.Getenv(key) == "" {
			return Config{}, fmt.Errorf("не задана переменная %s", key)
		}
	}
	if len(os.Getenv("JWT_SECRET")) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET должен содержать минимум 32 байта")
	}
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil || dbPort < 1 || dbPort > 65535 {
		return Config{}, fmt.Errorf("некорректный DB_PORT")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("некорректный PORT")
	}
	// URL безопасно экранирует специальные символы в реквизитах подключения.
	u := url.URL{Scheme: "postgres", User: url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")), Host: net.JoinHostPort(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")), Path: "/" + os.Getenv("DB_NAME")}
	q := u.Query()
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	q.Set("sslmode", sslmode)
	q.Set("connect_timeout", "5")
	u.RawQuery = q.Encode()
	return Config{DatabaseURL: u.String(), JWTSecret: os.Getenv("JWT_SECRET"), Port: port}, nil
}
