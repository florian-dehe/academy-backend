package identity

import "context"

// StaticResolver resolves tokens from a fixed map. For tests and APP_ENV=dev only.
type StaticResolver map[string]User

func (s StaticResolver) Resolve(_ context.Context, token string) (User, error) {
	if u, ok := s[token]; ok {
		return u, nil
	}
	return User{}, ErrInvalidToken
}

// UnconfiguredResolver rejects every token. Used outside dev when
// JOBBOARD_BASE_URL is not set.
type UnconfiguredResolver struct{}

func (UnconfiguredResolver) Resolve(context.Context, string) (User, error) {
	return User{}, ErrInvalidToken
}
