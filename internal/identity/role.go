// Package identity is the user module: roles, Laravel token auth middleware,
// the users store (sqlc) and the POST /me handler. Routes are registered in
// internal/app.
package identity

type Role string

const (
	RoleLearner       Role = "learner"
	RoleContentEditor Role = "content_editor"
	RoleAdmin         Role = "admin"
)

func (r Role) rank() int {
	switch r {
	case RoleLearner:
		return 1
	case RoleContentEditor:
		return 2
	case RoleAdmin:
		return 3
	}
	return 0
}

// AtLeast reports whether r is minRole or higher in admin ⊃ content_editor ⊃ learner.
func (r Role) AtLeast(minRole Role) bool { return r.rank() >= minRole.rank() && r.rank() > 0 }
