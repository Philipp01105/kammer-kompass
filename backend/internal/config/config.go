package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

type SessionConfig struct {
	CookieName string
	AuthKey    []byte
	EncKey     []byte
	Secure     bool
}

type Config struct {
	HTTPAddr    string
	LogLevel    slog.Level
	DatabaseURL string
	RedisAddr   string
	SecretSalt  string
	Session     SessionConfig
}

func FromEnv() (Config, error) {
	httpAddr := getenv("HTTP_ADDR", ":8080")
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	secretSalt := os.Getenv("SERVER_SECRET_SALT")
	if secretSalt == "" {
		return Config{}, errors.New("SERVER_SECRET_SALT is required")
	}

	logLevel, err := parseLogLevel(getenv("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}

	sessionAuthKey, err := mustBase64("SESSION_AUTH_KEY")
	if err != nil {
		return Config{}, err
	}
	sessionEncKey, err := mustBase64("SESSION_ENC_KEY")
	if err != nil {
		return Config{}, err
	}

	sessionCookieName := getenv("SESSION_COOKIE_NAME", "kk_session")
	sessionSecure := getenv("SESSION_SECURE", "false") == "true"

	return Config{
		HTTPAddr:    httpAddr,
		LogLevel:    logLevel,
		DatabaseURL: databaseURL,
		RedisAddr:   redisAddr,
		SecretSalt:  secretSalt,
		Session: SessionConfig{
			CookieName: sessionCookieName,
			AuthKey:    sessionAuthKey,
			EncKey:     sessionEncKey,
			Secure:     sessionSecure,
		},
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLogLevel(v string) (slog.Level, error) {
	switch v {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown LOG_LEVEL %q", v)
	}
}

func mustBase64(envKey string) ([]byte, error) {
	raw := os.Getenv(envKey)
	if raw == "" {
		return nil, fmt.Errorf("%s is required (base64)", envKey)
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be base64: %w", envKey, err)
	}
	return decoded, nil
}
