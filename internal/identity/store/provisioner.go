package store

import (
	"context"

	"github.com/remojii/academy-api/internal/identity"
)

// Provisioner implements identity.UserProvisioner on top of Store.
type Provisioner struct{ S *Store }

func (p Provisioner) ProvisionJobBoardUser(ctx context.Context, jb identity.JobBoardUser) (identity.User, error) {
	id := jb.ID
	u, err := p.S.UpsertByJobBoardID(ctx, CreateUserParams{Email: jb.Email, JobBoardUserID: &id, DisplayName: jb.DisplayName})
	if err != nil {
		return identity.User{}, err
	}
	return toIdentity(u), nil
}

func toIdentity(u User) identity.User {
	return identity.User{
		ID: u.ID, Email: u.Email, JobBoardUserID: u.JobBoardUserID,
		DisplayName: u.DisplayName, Role: identity.Role(u.Role),
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}
