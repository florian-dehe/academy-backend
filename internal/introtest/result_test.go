package introtest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/identity"
	"github.com/remojii/academy-api/internal/introtest"
)

type memStore struct {
	mu     sync.Mutex
	rows   []introtest.Result
	byUser []uuid.UUID
}

func (m *memStore) Create(_ context.Context, userID uuid.UUID, s introtest.Scores) (introtest.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := introtest.Result{ID: uuid.New(), Scores: s, CreatedAt: time.Now().UTC().Add(time.Duration(len(m.rows)) * time.Second)}
	m.rows = append(m.rows, r)
	m.byUser = append(m.byUser, userID)
	return r, nil
}

func (m *memStore) Latest(_ context.Context, userID uuid.UUID) (introtest.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.rows) - 1; i >= 0; i-- {
		if m.byUser[i] == userID {
			return m.rows[i], nil
		}
	}
	return introtest.Result{}, introtest.ErrNoResult
}

var me = identity.User{ID: uuid.New(), Role: identity.RoleLearner}

func router(s introtest.ResultStore) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-User") == "other" {
				next.ServeHTTP(w, r.WithContext(identity.WithUser(r.Context(), identity.User{ID: uuid.New()})))
				return
			}
			next.ServeHTTP(w, r.WithContext(identity.WithUser(r.Context(), me)))
		})
	})
	r.Post("/intro-test/result", introtest.Submit(s))
	r.Get("/me/personality-result", introtest.GetLatest(s))
	return r
}

func do(h http.Handler, method, path, body, user string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	req.Header.Set("X-User", user)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const valid = `{"openness":4.2,"conscientiousness":3,"extraversion":1,"agreeableness":5,"neuroticism":2.5}`

func TestSubmitAndGetLatest(t *testing.T) {
	h := router(&memStore{})

	require.Equal(t, http.StatusNotFound, do(h, "GET", "/me/personality-result", "", "").Code, "no result yet")

	rec := do(h, "POST", "/intro-test/result", valid, "")
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"openness":4.2`)
	require.Contains(t, rec.Body.String(), `"created_at"`)

	rec = do(h, "GET", "/me/personality-result", "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"neuroticism":2.5`)
}

func TestRetakeReplacesLatest(t *testing.T) {
	h := router(&memStore{})
	require.Equal(t, http.StatusCreated, do(h, "POST", "/intro-test/result", valid, "").Code)
	second := `{"openness":1,"conscientiousness":1,"extraversion":1,"agreeableness":1,"neuroticism":1}`
	require.Equal(t, http.StatusCreated, do(h, "POST", "/intro-test/result", second, "").Code)

	rec := do(h, "GET", "/me/personality-result", "", "")
	require.Contains(t, rec.Body.String(), `"openness":1,`)
}

func TestResultsAreOwnOnly(t *testing.T) {
	h := router(&memStore{})
	require.Equal(t, http.StatusCreated, do(h, "POST", "/intro-test/result", valid, "").Code)
	require.Equal(t, http.StatusNotFound, do(h, "GET", "/me/personality-result", "", "other").Code)
}

func TestSubmitValidation(t *testing.T) {
	h := router(&memStore{})
	for name, body := range map[string]string{
		"empty":         `{}`,
		"missing axis":  `{"openness":4,"conscientiousness":3,"extraversion":1,"agreeableness":5}`,
		"below range":   `{"openness":0.9,"conscientiousness":3,"extraversion":1,"agreeableness":5,"neuroticism":2}`,
		"above range":   `{"openness":5.1,"conscientiousness":3,"extraversion":1,"agreeableness":5,"neuroticism":2}`,
		"not a number":  `{"openness":"4","conscientiousness":3,"extraversion":1,"agreeableness":5,"neuroticism":2}`,
		"null score":    `{"openness":null,"conscientiousness":3,"extraversion":1,"agreeableness":5,"neuroticism":2}`,
		"unknown field": `{"openness":4,"conscientiousness":3,"extraversion":1,"agreeableness":5,"neuroticism":2,"answers":[1]}`,
		"not json":      `nope`,
	} {
		rec := do(h, "POST", "/intro-test/result", body, "")
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
	require.Equal(t, http.StatusCreated, do(h, "POST", "/intro-test/result",
		`{"openness":1,"conscientiousness":5,"extraversion":1,"agreeableness":5,"neuroticism":1}`, "").Code, "bounds are inclusive")
}
