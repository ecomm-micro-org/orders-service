package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid auth token")

type AuthManager struct {
	secretKey []byte
}

func NewAuthManager(secretKey string) (*AuthManager, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("secret key cannot be empty")
	}

	return &AuthManager{
		secretKey: []byte(secretKey),
	}, nil
}

func (am *AuthManager) ValidateToken(_ context.Context, tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}

		return am.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
