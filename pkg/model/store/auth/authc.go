// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
