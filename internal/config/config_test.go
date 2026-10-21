package config

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_TIMEOUT", "")
	t.Setenv("TIMEZONE", "")
	t.Setenv("SEED_DEMO_USER", "")

	t.Run("requires database URL", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("HTTP_ADDR", "")
		t.Setenv("TELEGRAM_BOT_TOKEN", "")

		if _, err := New(); err == nil {
			t.Fatal("expected an error when DATABASE_URL is empty")
		}
	})

	t.Run("uses default HTTP address", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("HTTP_ADDR", "")
		t.Setenv("TELEGRAM_BOT_TOKEN", "")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.HTTP().Addr() != ":8083" {
			t.Fatalf("expected default HTTP address :8083, got %q", cfg.HTTP().Addr())
		}
	})

	t.Run("uses custom HTTP address", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("HTTP_ADDR", "127.0.0.1:9000")
		t.Setenv("TELEGRAM_BOT_TOKEN", "")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.HTTP().Addr() != "127.0.0.1:9000" {
			t.Fatalf("expected custom HTTP address, got %q", cfg.HTTP().Addr())
		}
		if cfg.Postgres().URI() != "postgres://example" {
			t.Fatalf("unexpected database URI %q", cfg.Postgres().URI())
		}
	})

	t.Run("telegram is disabled without a token", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("TELEGRAM_BOT_TOKEN", "   ")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.Telegram().Enabled() {
			t.Fatal("expected telegram bot to be disabled")
		}
	})

	t.Run("telegram is enabled with a token", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("TELEGRAM_BOT_TOKEN", "123456:token")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.Telegram().Enabled() {
			t.Fatal("expected telegram bot to be enabled")
		}
		if cfg.Telegram().Token() != "123456:token" {
			t.Fatalf("unexpected token %q", cfg.Telegram().Token())
		}
	})

	t.Run("llm is disabled without a key", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.LLM().Enabled() {
			t.Fatal("expected LLM to be disabled")
		}
		if cfg.LLM().Model() != "openai/gpt-oss-120b" {
			t.Fatalf("unexpected default LLM model %q", cfg.LLM().Model())
		}
		if cfg.LLM().BaseURL() != "https://api.groq.com/openai/v1" {
			t.Fatalf("unexpected default LLM base URL %q", cfg.LLM().BaseURL())
		}
		if cfg.LLM().Timeout() != 20*time.Second {
			t.Fatalf("unexpected default LLM timeout %s", cfg.LLM().Timeout())
		}
	})

	t.Run("llm uses custom settings", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("GROQ_API_KEY", "gsk_test")
		t.Setenv("LLM_MODEL", "qwen/qwen3.8-27b")
		t.Setenv("LLM_BASE_URL", "https://example.com/v1/")
		t.Setenv("LLM_TIMEOUT", "5s")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.LLM().Enabled() {
			t.Fatal("expected LLM to be enabled")
		}
		if cfg.LLM().Model() != "qwen/qwen3.8-27b" {
			t.Fatalf("unexpected LLM model %q", cfg.LLM().Model())
		}
		if cfg.LLM().BaseURL() != "https://example.com/v1" {
			t.Fatalf("unexpected LLM base URL %q", cfg.LLM().BaseURL())
		}
		if cfg.LLM().Timeout() != 5*time.Second {
			t.Fatalf("unexpected LLM timeout %s", cfg.LLM().Timeout())
		}
	})

	t.Run("rejects invalid LLM timeout", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("LLM_TIMEOUT", "soon")

		if _, err := New(); err == nil {
			t.Fatal("expected an error for an invalid LLM timeout")
		}
	})

	t.Run("rejects invalid LLM base URL", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("LLM_BASE_URL", "not-a-url")

		if _, err := New(); err == nil {
			t.Fatal("expected an error for an invalid LLM base URL")
		}
	})

	t.Run("uses default timezone", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.TimeZone().String() != "Europe/Moscow" {
			t.Fatalf("unexpected timezone %q", cfg.TimeZone())
		}
	})

	t.Run("uses custom timezone", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("TIMEZONE", "UTC")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.TimeZone().String() != "UTC" {
			t.Fatalf("unexpected timezone %q", cfg.TimeZone())
		}
	})

	t.Run("rejects invalid timezone", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("TIMEZONE", "Mars/Olympus")

		if _, err := New(); err == nil {
			t.Fatal("expected an error for an invalid timezone")
		}
	})

	t.Run("demo user is disabled by default", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.SeedDemoUser() {
			t.Fatal("expected demo user seeding to be disabled")
		}
	})

	t.Run("demo user can be enabled", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("SEED_DEMO_USER", "true")

		cfg, err := New()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.SeedDemoUser() {
			t.Fatal("expected demo user seeding to be enabled")
		}
	})

	t.Run("rejects invalid demo user flag", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("SEED_DEMO_USER", "maybe")

		if _, err := New(); err == nil {
			t.Fatal("expected an error for an invalid SEED_DEMO_USER")
		}
	})
}
