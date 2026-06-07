package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthManagerRejectsEmptySecret(t *testing.T) {
	_, err := NewAuthManager("")
	require.Error(t, err)
}

func TestValidateTokenSuccess(t *testing.T) {
	am, err := NewAuthManager("secret")
	require.NoError(t, err)

	claims := NewUserClaims("user-1", "user@example.com", "user", "seller", time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("secret"))
	require.NoError(t, err)

	got, err := am.ValidateToken(context.Background(), tokenStr)
	require.NoError(t, err)
	assert.Equal(t, claims.UserID, got.UserID)
	assert.Equal(t, claims.Email, got.Email)
	assert.Equal(t, claims.Role, got.Role)
}

func TestValidateTokenRejectsMalformedToken(t *testing.T) {
	am, err := NewAuthManager("secret")
	require.NoError(t, err)

	_, err = am.ValidateToken(context.Background(), "not-a-jwt")
	require.Error(t, err)
}

func TestValidateTokenRejectsWrongSecret(t *testing.T) {
	am, err := NewAuthManager("secret")
	require.NoError(t, err)

	claims := NewUserClaims("user-1", "user@example.com", "user", "seller", time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("other-secret"))
	require.NoError(t, err)

	_, err = am.ValidateToken(context.Background(), tokenStr)
	require.Error(t, err)
}
