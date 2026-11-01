package identity_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/identity"
)

type fakeProv struct{ calls atomic.Int32 }

func (f *fakeProv) ProvisionJobBoardUser(_ context.Context, jb identity.JobBoardUser) (identity.User, error) {
	f.calls.Add(1)
	return identity.User{ID: uuid.New(), Email: jb.Email, Role: identity.RoleLearner}, nil
}

func newResolver(t *testing.T, h http.HandlerFunc) (*identity.JobBoardResolver, *fakeProv) {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	p := &fakeProv{}
	r, err := identity.NewJobBoardResolver(identity.JobBoardConfig{BaseURL: srv.URL}, p)
	require.NoError(t, err)
	return r, p
}

func TestJobBoardResolver(t *testing.T) {
	var hits atomic.Int32
	r, prov := newResolver(t, func(w http.ResponseWriter, req *http.Request) {
		hits.Add(1)
		switch req.Header.Get("Authorization") {
		case "Bearer 1|good":
			_, _ = w.Write([]byte(`{"data":{"id":42,"email":"A@Example.com","name":"Ada"}}`))
		case "Bearer 1|boom":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	ctx := context.Background()

	u, err := r.Resolve(ctx, "1|good")
	require.NoError(t, err)
	require.Equal(t, "a@example.com", u.Email)
	_, err = r.Resolve(ctx, "1|good")
	require.NoError(t, err)
	require.EqualValues(t, 1, hits.Load(), "second call served from cache")
	require.EqualValues(t, 1, prov.calls.Load())

	_, err = r.Resolve(ctx, "1|bad")
	require.ErrorIs(t, err, identity.ErrInvalidToken)
	_, err = r.Resolve(ctx, "1|bad")
	require.ErrorIs(t, err, identity.ErrInvalidToken)
	require.EqualValues(t, 2, hits.Load(), "401 cached briefly")

	_, err = r.Resolve(ctx, "1|boom")
	require.Error(t, err)
	require.NotErrorIs(t, err, identity.ErrInvalidToken, "upstream 5xx is not a 401")
}
