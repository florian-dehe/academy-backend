// Package identity holds the Role type. Authentication lives in internal/app;
// the users table is queried through identity/store (sqlc).
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
