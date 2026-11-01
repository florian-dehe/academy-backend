package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/remojii/academy-api/api"
)

type specDoc struct {
	Paths map[string]map[string]any `yaml:"paths"`
}

// The spec and the router must list the same operations, and x-min-role must
// match what the role guards actually enforce.
func TestOpenAPIMatchesRouter(t *testing.T) {
	var doc specDoc
	require.NoError(t, yaml.Unmarshal(api.Spec, &doc))

	want := map[string]string{} // "METHOD /path" -> min role
	for p, item := range doc.Paths {
		for k, v := range item {
			if k == "parameters" {
				continue
			}
			op := v.(map[string]any)
			want[strings.ToUpper(k)+" "+p], _ = op["x-min-role"].(string)
		}
	}

	srv := newServer(t)
	got := map[string]bool{}
	skip := map[string]bool{"/openapi.yaml": true, "/docs": true}
	err := chi.Walk(srv.(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimSuffix(route, "/*")
		if skip[route] {
			return nil
		}
		key := method + " " + route
		got[key] = true
		role, ok := want[key]
		require.True(t, ok, "route %s missing from api/openapi.yaml", key)
		require.Equal(t, role, probeMinRole(t.Context(), srv, method, route), "x-min-role mismatch for %s", key)
		return nil
	})
	require.NoError(t, err)
	for key := range want {
		require.True(t, got[key], "spec operation %s has no route", key)
	}
}

var paramRE = regexp.MustCompile(`\{\w+\}`)

// probeMinRole returns the lowest role (or "none") that gets past the guards.
func probeMinRole(ctx context.Context, h http.Handler, method, route string) string {
	path := paramRE.ReplaceAllString(route, "x")
	for _, tok := range []string{"", "learner", "editor", "admin"} {
		req := httptest.NewRequestWithContext(ctx, method, path, nil)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
			if tok == "" {
				return "none"
			}
			return tok
		}
	}
	return "denied"
}

func TestDocsEndpoints(t *testing.T) {
	srv := newServer(t)
	for path, ct := range map[string]string{"/openapi.yaml": "application/yaml", "/docs": "text/html"} {
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		require.Equal(t, 200, w.Code, path)
		require.Contains(t, w.Header().Get("Content-Type"), ct, path)
	}
}
