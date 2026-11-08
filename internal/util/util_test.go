package util_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/util"
)

func TestRecovererReturnsJSON(t *testing.T) {
	h := util.Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	require.Contains(t, rec.Body.String(), `"code":"internal"`)
}

func TestRequestLoggerIncludesActor(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	h := util.RequestLogger(l)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		util.SetActor(r.Context(), "u1", "admin")
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v1/levels", nil))
	require.Contains(t, buf.String(), `"user_id":"u1"`)
	require.Contains(t, buf.String(), `"role":"admin"`)
}

func TestDecodeLimitsAndTrailing(t *testing.T) {
	var v struct{ A int }
	req := func(body string) *http.Request {
		return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
	}
	require.NoError(t, util.Decode(req(`{"A":1}`), &v))
	require.Error(t, util.Decode(req(`{"A":1} {"A":2}`), &v))
	require.Error(t, util.Decode(req(`{"B":1}`), &v))
	big := `{"A":1,"pad":"` + strings.Repeat("x", util.MaxBodyBytes) + `"}`
	require.Error(t, util.Decode(req(big), &v))
}
