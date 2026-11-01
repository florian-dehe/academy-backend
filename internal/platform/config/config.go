// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// Env has no default on purpose: forgetting it must not enable dev auth.
	Env         string `env:"APP_ENV,required"` // dev | staging | prod
	HTTPAddr    string `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	// AutoMigrate runs GORM AutoMigrate (internal/schema) at startup.
	// Off by default; .env turns it on for local dev. Move to versioned
	// migrations before production.
	AutoMigrate bool `env:"AUTO_MIGRATE" envDefault:"false"`
	// Job Board (Laravel Sanctum) is the only authentication system. Academy
	// validates bearer tokens by calling its current-user endpoint.
	JobBoardBaseURL     string `env:"JOBBOARD_BASE_URL"`
	JobBoardMePath      string `env:"JOBBOARD_ME_PATH" envDefault:"/api/me"`
	JobBoardCacheTTLSec int    `env:"JOBBOARD_CACHE_TTL_SECONDS" envDefault:"60"`
}

func (c Config) IsDev() bool { return c.Env == "dev" }

func Load() (Config, error) {
	c, err := env.ParseAs[Config]()
	if err != nil {
		return c, err
	}
	switch c.Env {
	case "dev", "staging", "prod":
	default:
		return c, fmt.Errorf("config: APP_ENV must be dev, staging or prod, got %q", c.Env)
	}
	return c, nil
}

// NewLogger builds the JSON logger shared by the API and the worker.
func (c Config) NewLogger() (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		return nil, fmt.Errorf("config: LOG_LEVEL %q: %w", c.LogLevel, err)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})), nil
}
