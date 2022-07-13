package auth

import (
	"context"
	"fmt"

	"kubegems.io/kubegems/pkg/utils/jwt"
)

type UserInfo struct {
	Username string `json:"username,omitempty"`
}

type AuthenticationManager interface {
	UserInfo(ctx context.Context, token string) (UserInfo, error)
}

func NewJWTAuthenticationManager(opt *jwt.Options) *JWTAuthenticationManager {
	if opt == nil {
		opt = jwt.DefaultOptions()
	}
	return &JWTAuthenticationManager{jwtparser: opt.ToJWT()}
}

type JWTAuthenticationManager struct {
	jwtparser *jwt.JWT
}

func (a *JWTAuthenticationManager) UserInfo(ctx context.Context, token string) (UserInfo, error) {
	cliams, err := a.jwtparser.ParseToken(token)
	if err != nil {
		return UserInfo{}, err
	}
	username := cliams.Subject
	if username == "" {
		if payload, ok := cliams.Payload.(map[string]interface{}); ok {
			for _, key := range []string{"username", "Username", "sub"} {
				if val, ok := payload[key].(string); ok && val != "" {
					username = val
					break
				}
			}
		}
	}
	if username == "" {
		return UserInfo{}, fmt.Errorf("username not found in token")
	}
	return UserInfo{Username: username}, nil
}
