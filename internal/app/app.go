// Package app wires modules into one HTTP router.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/remojii/academy-api/api"
	"github.com/remojii/academy-api/internal/util"
)

type Deps struct {
	Logger *slog.Logger
	// Ready reports dependency health for /readyz (e.g. DB ping). Optional.
	Ready func(ctx context.Context) error
}

type Config struct {
	// Env has no default on purpose: forgetting it must not enable dev auth.
	Env         string `env:"APP_ENV,required"` // dev | staging | prod
	HTTPAddr    string `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, util.RequestLogger(d.Logger), util.Recoverer)
	r.NotFound(util.NotFound)
	r.MethodNotAllowed(util.MethodNotAllowed)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		util.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/openapi.yaml", util.SpecHandler(api.Spec))
	r.Get("/docs", util.SwaggerUI)

	r.Route("/v1", func(r chi.Router) {
		routes(r)
	})
	return r
}

func (c Config) IsDev() bool { return c.Env == "dev" }

func LoadConfig() (Config, error) {
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

// JsonLogger builds the JSON logger shared by the API and the worker.
func (c Config) JsonLogger() (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		return nil, fmt.Errorf("config: LOG_LEVEL %q: %w", c.LogLevel, err)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})), nil
}
