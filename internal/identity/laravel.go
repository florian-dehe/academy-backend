package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// LaravelUser is the identity the Laravel app reports for a bearer token.
type LaravelUser struct {
	ID    string
	Email string
	Name  string
}

// Errors returned by UserClient.
var (
	// ErrTokenRejected: Laravel does not accept the token (expired, revoked, unknown).
	ErrTokenRejected = errors.New("identity: token rejected")
	// ErrUpstream: Laravel could not answer (network, timeout, 5xx, 429, bad body).
	ErrUpstream = errors.New("identity: upstream unavailable")
)

// UserClient resolves a Sanctum access token to its Laravel user. It is an
// interface so tests can fake it and a short-TTL cache can wrap it later.
type UserClient interface {
	CurrentUser(ctx context.Context, token string) (LaravelUser, error)
}

// UserClientFunc adapts a function to UserClient (handy for tests).
type UserClientFunc func(ctx context.Context, token string) (LaravelUser, error)

func (f UserClientFunc) CurrentUser(ctx context.Context, token string) (LaravelUser, error) {
	return f(ctx, token)
}

// HTTPUserClient calls the Laravel current-user endpoint over REST.
type HTTPUserClient struct {
	endpoint string
	http     *http.Client
}

// NewHTTPUserClient builds a client for baseURL+path with a per-request timeout.
func NewHTTPUserClient(baseURL, path string, timeout time.Duration) (*HTTPUserClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("identity: invalid Laravel base URL %q", baseURL)
	}
	return &HTTPUserClient{
		endpoint: strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/"),
		http:     &http.Client{Timeout: timeout},
	}, nil
}

// CurrentUser expects {"data": {"id": ..., "email": ..., "first_name": ..., "last_name": ...}}.
func (c *HTTPUserClient) CurrentUser(ctx context.Context, token string) (LaravelUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, http.NoBody)
	if err != nil {
		return LaravelUser{}, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return LaravelUser{}, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return LaravelUser{}, ErrTokenRejected
	default: // includes 404: a wrong LARAVEL_USER_PATH must not log everyone out
		return LaravelUser{}, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var body struct {
		Data struct {
			ID        json.RawMessage `json:"id"`
			Email     string          `json:"email"`
			FirstName string          `json:"first_name"`
			LastName  string          `json:"last_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&body); err != nil {
		return LaravelUser{}, fmt.Errorf("%w: decode: %w", ErrUpstream, err)
	}
	id, ok := parseID(body.Data.ID)
	if !ok || body.Data.Email == "" {
		return LaravelUser{}, fmt.Errorf("%w: response lacks a valid id or email", ErrUpstream)
	}
	return LaravelUser{
		ID:    id,
		Email: body.Data.Email,
		Name:  strings.TrimSpace(body.Data.FirstName + " " + body.Data.LastName),
	}, nil
}

// parseID accepts a non-empty JSON string or an integer, nothing else.
func parseID(raw json.RawMessage) (string, bool) {
	var str string
	if json.Unmarshal(raw, &str) == nil {
		return str, str != ""
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return strconv.FormatInt(n, 10), true
	}
	return "", false
}
