package identity_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/remojii/academy-api/internal/identity"
)

func TestRoleAtLeast(t *testing.T) {
	require.True(t, identity.RoleAdmin.AtLeast(identity.RoleContentEditor))
	require.True(t, identity.RoleContentEditor.AtLeast(identity.RoleLearner))
	require.True(t, identity.RoleLearner.AtLeast(identity.RoleLearner))
	require.False(t, identity.RoleLearner.AtLeast(identity.RoleContentEditor))
	require.False(t, identity.RoleContentEditor.AtLeast(identity.RoleAdmin))
	require.False(t, identity.Role("").AtLeast(identity.RoleLearner))
}
