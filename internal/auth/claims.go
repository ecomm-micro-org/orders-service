package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserClaims struct {
	UserID   string `json:"id"`
	Email    string
	Username string
	Role     string
	jwt.RegisteredClaims
}

func NewUserClaims(userID, email, username, role string, duration time.Duration) *UserClaims {
	tokenID := uuid.NewString()

	return &UserClaims{
		UserID:   userID,
		Email:    email,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(duration))),
		},
	}
}
