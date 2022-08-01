package auth

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt"
)

type UserInfo struct {
	Username string `json:"username,omitempty"`
}

type AuthenticationManager interface {
	UserInfo(ctx context.Context, token string) (UserInfo, error)
}

func NewUnVerifyJWTAuthenticationManager() *UnVerifyJWTAuthenticationManager {
	return &UnVerifyJWTAuthenticationManager{}
}

type UnVerifyJWTAuthenticationManager struct{}

func (a *UnVerifyJWTAuthenticationManager) UserInfo(ctx context.Context, token string) (UserInfo, error) {
	claims := &jwt.StandardClaims{}
	// Do not validate signature, because we do not know how to verify it now.
	_, _, err := (&jwt.Parser{}).ParseUnverified(token, claims)
	if err != nil {
		return UserInfo{}, fmt.Errorf("parse token: %v", err)
	}
	username := claims.Subject
	if username == "" {
		return UserInfo{}, fmt.Errorf("sub not found in token")
	}
	return UserInfo{Username: username}, nil
}
