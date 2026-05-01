package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        int
	DatabaseURL string
	LogLevel    string // например, "debug" или "info"
	LogFormat   string // "text" или "json"
	JWTSecret   string
	JWTTokenTTL time.Duration
}

// Load читает .env (если есть) и возвращает заполненную структуру.
func Load() (*Config, error) {
	// Игнорируем ошибку, если файла нет – это нормально для production
	_ = godotenv.Load()

	dbURL, err := getEnvOrError("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        getEnvAsInt("PORT", 8080),
		DatabaseURL: dbURL,
	}

	if cfg.Port < 1024 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", cfg.Port)
	}

	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.LogFormat = getEnv("LOG_FORMAT", "text")

	cfg.JWTSecret, err = getEnvOrError("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	ttlStr := getEnv("JWT_TOKEN_TTL", "24h")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_TOKEN_TTL: %w", err)
	}
	cfg.JWTTokenTTL = ttl

	return cfg, nil
}

// Вспомогательная функция: читает переменную или возвращает дефолт.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// Для чисел
func getEnvAsInt(key string, defaultVal int) int {
	strVal := getEnv(key, "")
	if strVal == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(strVal)
	if err != nil {
		fmt.Printf("WARN: invalid integer for %s: %q. Using default %d\n", key, strVal, defaultVal)
		return defaultVal
	}
	return val
}

// Для обязательных переменных
func getEnvOrError(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", fmt.Errorf("FATAL: environment variable %s is required", key)
	}
	return val, nil
}
