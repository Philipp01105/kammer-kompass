package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	key32 := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("SERVER_SECRET_SALT", "0123456789abcdef0123456789abcdef")
	t.Setenv("SESSION_AUTH_KEY", key32)
	t.Setenv("SESSION_ENC_KEY", key32)
}

func TestFromEnvRejectsProductionWithInsecureSession(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECURE", "false")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "SESSION_SECURE=true") {
		t.Fatalf("expected production SESSION_SECURE error, got %v", err)
	}
}

func TestFromEnvRejectsProductionBootstrap(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECURE", "true")
	t.Setenv("BOOTSTRAP_SUPER_ADMIN", "true")
	t.Setenv("BOOTSTRAP_SUPER_ADMIN_PASSWORD", "local-only-bootstrap-password")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "BOOTSTRAP_SUPER_ADMIN must be false") {
		t.Fatalf("expected production bootstrap error, got %v", err)
	}
}

func TestFromEnvRejectsProductionLocalhostCORS(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECURE", "true")
	t.Setenv("BOOTSTRAP_SUPER_ADMIN", "false")
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "invalid production origin") {
		t.Fatalf("expected production CORS error, got %v", err)
	}
}

func TestFromEnvRequiresBootstrapPasswordWhenEnabled(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "development")
	t.Setenv("BOOTSTRAP_SUPER_ADMIN", "true")
	t.Setenv("BOOTSTRAP_SUPER_ADMIN_PASSWORD", "")

	_, err := FromEnv()
	if err == nil || !strings.Contains(err.Error(), "BOOTSTRAP_SUPER_ADMIN_PASSWORD is required") {
		t.Fatalf("expected bootstrap password error, got %v", err)
	}
}

func TestFromEnvAcceptsProductionSafeConfig(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECURE", "true")
	t.Setenv("BOOTSTRAP_SUPER_ADMIN", "false")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("expected safe production config, got %v", err)
	}
	if cfg.AppEnv != "production" || !cfg.Session.Secure || cfg.BootstrapSuperAdmin {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
