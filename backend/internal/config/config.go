package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
)

type SessionConfig struct {
	CookieName string
	AuthKey    []byte
	EncKey     []byte
	Secure     bool
}

type Config struct {
	HTTPAddr            string
	AppEnv              string
	LogLevel            slog.Level
	DatabaseURL         string
	RedisAddr           string
	SecretSalt          string
	Session             SessionConfig
	BootstrapSuperAdmin bool
	BootstrapPassword   string
	AllowedOrigins      []string
	TrustedProxies      []string
}

func FromEnv() (Config, error) {
	appEnv := strings.ToLower(getenv("APP_ENV", "development"))
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
	bootstrapSuperAdmin := getenv("BOOTSTRAP_SUPER_ADMIN", "false") == "true"
	bootstrapPassword := os.Getenv("BOOTSTRAP_SUPER_ADMIN_PASSWORD")

	allowedOrigins := splitTrimmed(getenv("ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"), ",")
	trustedProxies := splitTrimmed(os.Getenv("TRUSTED_PROXIES"), ",")

	cfg := Config{
		HTTPAddr:            httpAddr,
		AppEnv:              appEnv,
		LogLevel:            logLevel,
		DatabaseURL:         databaseURL,
		RedisAddr:           redisAddr,
		SecretSalt:          secretSalt,
		BootstrapSuperAdmin: bootstrapSuperAdmin,
		BootstrapPassword:   bootstrapPassword,
		AllowedOrigins:      allowedOrigins,
		TrustedProxies:      trustedProxies,
		Session: SessionConfig{
			CookieName: sessionCookieName,
			AuthKey:    sessionAuthKey,
			EncKey:     sessionEncKey,
			Secure:     sessionSecure,
		},
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
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

func splitTrimmed(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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

func validate(cfg Config) error {
	if cfg.BootstrapSuperAdmin && cfg.BootstrapPassword == "" {
		return errors.New("BOOTSTRAP_SUPER_ADMIN_PASSWORD is required when BOOTSTRAP_SUPER_ADMIN=true")
	}
	if len(cfg.BootstrapPassword) > 0 && len(cfg.BootstrapPassword) < 16 {
		return errors.New("BOOTSTRAP_SUPER_ADMIN_PASSWORD must be at least 16 characters")
	}
	if cfg.AppEnv != "production" {
		return nil
	}
	if !cfg.Session.Secure {
		return errors.New("SESSION_SECURE=true is required in production")
	}
	if cfg.BootstrapSuperAdmin {
		return errors.New("BOOTSTRAP_SUPER_ADMIN must be false in production")
	}
	if len(cfg.SecretSalt) < 32 {
		return errors.New("SERVER_SECRET_SALT must be at least 32 characters in production")
	}
	if len(cfg.AllowedOrigins) == 0 {
		return errors.New("ALLOWED_ORIGINS must contain at least one HTTPS origin in production")
	}
	for _, origin := range cfg.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" {
			return fmt.Errorf("ALLOWED_ORIGINS contains invalid production origin %q", origin)
		}
		host := strings.ToLower(u.Hostname())
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return fmt.Errorf("ALLOWED_ORIGINS must not contain localhost origin %q in production", origin)
		}
	}
	return nil
}
