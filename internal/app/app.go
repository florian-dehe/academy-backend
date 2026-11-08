// Package app wires modules into one HTTP router.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"github.com/remojii/academy-api/api"
	"github.com/remojii/academy-api/internal/httpx"
	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/introtest"
)

type Deps struct {
	Logger *slog.Logger
	// Ready reports dependency health for /readyz (e.g. DB ping). Optional.
	Ready func(ctx context.Context) error
	// LaravelUsers validates bearer tokens against the Laravel app.
	LaravelUsers identity.UserClient
	// Users stores Academy users.
	Users identity.UserStore
	// Results stores intro test (personality) results.
	Results introtest.ResultStore
}

type Config struct {
	// Env has no default on purpose: forgetting it must not enable dev auth.
	Env         string `env:"APP_ENV,required"` // dev | staging | prod
	HTTPAddr    string `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`

	// Laravel app that owns login and the Sanctum tokens. No default for the
	// URL: forgetting it must fail startup, never disable auth.
	LaravelBaseURL     string        `env:"LARAVEL_BASE_URL,required"`
	LaravelUserPath    string        `env:"LARAVEL_USER_PATH" envDefault:"/api/v1/user"`
	LaravelHTTPTimeout time.Duration `env:"LARAVEL_HTTP_TIMEOUT" envDefault:"5s"`
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	// httpx.Recoverer and RequestLogger replace chi's: ours return the JSON
	// error body and log the authenticated user.
	r.Use(middleware.RequestID, httpx.RequestLogger(d.Logger), httpx.Recoverer, middleware.Timeout(25*time.Second))
	r.NotFound(httpx.NotFound)
	r.MethodNotAllowed(httpx.MethodNotAllowed)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]string{"status": "ok"})
	})
	r.Get("/openapi.yaml", httpx.SpecHandler(api.Spec))
	r.Get("/docs", httpx.SwaggerUI)

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		routes(r, d)
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

// JSONLogger builds the JSON logger shared by the API and the worker.
func (c Config) JSONLogger() (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		return nil, fmt.Errorf("config: LOG_LEVEL %q: %w", c.LogLevel, err)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})), nil
}
