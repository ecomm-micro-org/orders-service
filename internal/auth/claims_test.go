package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserClaims(t *testing.T) {
	claims := NewUserClaims("user-1", "user@example.com", "risbern", "seller", time.Hour)

	require.NotNil(t, claims)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, "risbern", claims.Username)
	assert.Equal(t, "seller", claims.Role)
	assert.NotEmpty(t, claims.RegisteredClaims.ID)
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.After(claims.IssuedAt.Time))
}
