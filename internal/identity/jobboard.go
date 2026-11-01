package identity

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// JobBoardUser is the subset of the Job Board "current user" response Academy uses.
type JobBoardUser struct {
	ID          string
	Email       string
	DisplayName *string
}

// UserProvisioner maps a Job Board user to the local Academy user, creating it
// on first sight (role learner). Roles are owned by Academy, never by Job Board.
type UserProvisioner interface {
	ProvisionJobBoardUser(ctx context.Context, jb JobBoardUser) (User, error)
}

// JobBoardConfig configures JobBoardResolver.
type JobBoardConfig struct {
	BaseURL string // e.g. https://jobboard.example.com
	MePath  string // default /api/me
	// CacheTTL bounds how long a revoked token keeps working. Default 60s.
	CacheTTL time.Duration
	// NegativeTTL caches rejected tokens briefly to blunt guessing floods. Default 5s.
	NegativeTTL time.Duration
	HTTPClient  *http.Client // default: 3s timeout
}

// JobBoardResolver validates Laravel Sanctum access tokens by calling Job
// Board's current-user endpoint with the same bearer token. Job Board stays the
// only authentication system; Academy only provisions a local user row.
type JobBoardResolver struct {
	meURL string
	ttl   time.Duration
	negt  time.Duration
	hc    *http.Client
	prov  UserProvisioner
	now   func() time.Time

	sf    singleflight.Group
	mu    sync.Mutex
	cache map[[32]byte]cacheEntry
}

type cacheEntry struct {
	user    User
	invalid bool
	expires time.Time
}

const maxCacheEntries = 10_000

func NewJobBoardResolver(cfg JobBoardConfig, prov UserProvisioner) (*JobBoardResolver, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("identity: job board base url is empty")
	}
	if cfg.MePath == "" {
		cfg.MePath = "/api/me"
	}
	me, err := url.JoinPath(cfg.BaseURL, cfg.MePath)
	if err != nil {
		return nil, fmt.Errorf("identity: job board url: %w", err)
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 60 * time.Second
	}
	if cfg.NegativeTTL <= 0 {
		cfg.NegativeTTL = 5 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 3 * time.Second}
	}
	return &JobBoardResolver{
		meURL: me, ttl: cfg.CacheTTL, negt: cfg.NegativeTTL, hc: cfg.HTTPClient,
		prov: prov, now: time.Now, cache: map[[32]byte]cacheEntry{},
	}, nil
}

func (r *JobBoardResolver) Resolve(ctx context.Context, token string) (User, error) {
	key := sha256.Sum256([]byte(token))
	if e, ok := r.lookup(key); ok {
		if e.invalid {
			return User{}, ErrInvalidToken
		}
		return e.user, nil
	}
	// Detach from the first caller's cancellation so one client hanging up
	// does not fail every request waiting on the same token.
	v, err, _ := r.sf.Do(string(key[:]), func() (any, error) {
		return r.fetch(context.WithoutCancel(ctx), token, key)
	})
	if err != nil {
		return User{}, err
	}
	return v.(User), nil
}

func (r *JobBoardResolver) fetch(ctx context.Context, token string, key [32]byte) (User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.meURL, nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := r.hc.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("identity: job board: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		r.store(key, cacheEntry{invalid: true, expires: r.now().Add(r.negt)})
		return User{}, ErrInvalidToken
	case resp.StatusCode != http.StatusOK:
		// Not a verdict on the token: surface as 5xx, do not cache.
		return User{}, fmt.Errorf("identity: job board: unexpected status %d", resp.StatusCode)
	}

	jb, err := decodeJobBoardUser(resp)
	if err != nil {
		return User{}, err
	}
	u, err := r.prov.ProvisionJobBoardUser(ctx, jb)
	if err != nil {
		return User{}, fmt.Errorf("identity: provision: %w", err)
	}
	r.store(key, cacheEntry{user: u, expires: r.now().Add(r.ttl)})
	return u, nil
}

func (r *JobBoardResolver) lookup(key [32]byte) (cacheEntry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.cache[key]
	if !ok || !r.now().Before(e.expires) {
		delete(r.cache, key)
		return cacheEntry{}, false
	}
	return e, true
}

func (r *JobBoardResolver) store(key [32]byte, e cacheEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.cache) >= maxCacheEntries {
		now := r.now()
		for k, v := range r.cache {
			if !now.Before(v.expires) {
				delete(r.cache, k)
			}
		}
		if len(r.cache) >= maxCacheEntries {
			clear(r.cache)
		}
	}
	r.cache[key] = e
}

// flexString accepts a JSON string or number (Laravel ids are usually numeric).
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexString(n.String())
	return nil
}

// decodeJobBoardUser reads {id,email,name} either at top level or under "data"
// (Laravel API resources wrap by default).
// ASSUMPTION: field names id/email/name. Adjust when the Job Board contract is fixed.
func decodeJobBoardUser(resp *http.Response) (JobBoardUser, error) {
	type body struct {
		ID    flexString `json:"id"`
		Email string     `json:"email"`
		Name  *string    `json:"name"`
	}
	var raw struct {
		body
		Data *body `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return JobBoardUser{}, fmt.Errorf("identity: job board: decode: %w", err)
	}
	b := raw.body
	if raw.Data != nil {
		b = *raw.Data
	}
	id := strings.TrimSpace(string(b.ID))
	if id == "" || b.Email == "" {
		return JobBoardUser{}, errors.New("identity: job board: response missing id or email")
	}
	if _, err := strconv.ParseUint(id, 10, 64); err != nil && len(id) > 64 {
		return JobBoardUser{}, errors.New("identity: job board: bad id")
	}
	return JobBoardUser{ID: id, Email: strings.ToLower(b.Email), DisplayName: b.Name}, nil
}
