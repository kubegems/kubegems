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

package oidc

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/zitadel/oidc/pkg/oidc"
	"github.com/zitadel/oidc/pkg/op"
	"gopkg.in/square/go-jose.v2"
)

type LocalStorage struct {
	LocalOPStorage
	LocalAuthStorage
}

type OIDCOptions struct {
	Issuer   string
	CertFile string
	KeyFile  string
}

func NewLocalStorage(ctx context.Context, options *OIDCOptions) (*LocalStorage, error) {
	tlscert, err := tls.LoadX509KeyPair(options.CertFile, options.KeyFile)
	if err != nil {
		return nil, err
	}
	auth := LocalAuthStorage{
		Certs: tlscert,
		jwks: &jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				(&jose.JSONWebKey{
					Key:       tlscert.PrivateKey,
					KeyID:     "",
					Algorithm: string(jose.RS256),
					Use:       "sig",
				}).Public(),
			},
		},
		signkey: jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       tlscert.PrivateKey,
		},
	}
	return &LocalStorage{LocalAuthStorage: auth}, nil
}

func (s *LocalStorage) Health(context.Context) error {
	return nil
}

type LocalOPStorage struct{}

func (s LocalOPStorage) GetClientByClientID(ctx context.Context, clientID string) (op.Client, error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalOPStorage) AuthorizeClientIDSecret(ctx context.Context, clientID string, clientSecret string) error {
	panic("not implemented") // TODO: Implement
}

func (s LocalOPStorage) SetUserinfoFromScopes(ctx context.Context, userinfo oidc.UserInfoSetter, userID string, clientID string, scopes []string) error {
	panic("not implemented") // TODO: Implement
}

func (s LocalOPStorage) SetUserinfoFromToken(ctx context.Context, userinfo oidc.UserInfoSetter, tokenID string, subject string, origin string) error {
	panic("not implemented") // TODO: Implement
}

func (s LocalOPStorage) SetIntrospectionFromToken(
	ctx context.Context, userinfo oidc.IntrospectionResponse, tokenID string, subject string, clientID string,
) error {
	panic("not implemented") // TODO: Implement
}

func (s LocalOPStorage) GetPrivateClaimsFromScopes(ctx context.Context, userID string, clientID string, scopes []string) (map[string]interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalOPStorage) GetKeyByIDAndUserID(ctx context.Context, keyID string, userID string) (*jose.JSONWebKey, error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalOPStorage) ValidateJWTProfileScopes(ctx context.Context, userID string, scopes []string) ([]string, error) {
	panic("not implemented") // TODO: Implement
}

type LocalAuthStorage struct {
	Certs tls.Certificate

	signkey jose.SigningKey
	jwks    *jose.JSONWebKeySet
}

func (s LocalAuthStorage) CreateAuthRequest(_ context.Context, _ *oidc.AuthRequest, _ string) (op.AuthRequest, error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalAuthStorage) AuthRequestByID(_ context.Context, _ string) (op.AuthRequest, error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalAuthStorage) AuthRequestByCode(_ context.Context, _ string) (op.AuthRequest, error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalAuthStorage) SaveAuthCode(_ context.Context, _ string, _ string) error {
	panic("not implemented") // TODO: Implement
}

func (s LocalAuthStorage) DeleteAuthRequest(_ context.Context, _ string) error {
	panic("not implemented") // TODO: Implement
}

// The TokenRequest parameter of CreateAccessToken can be any of:
//
// * TokenRequest as returned by ClientCredentialsStorage.ClientCredentialsTokenRequest,
//
// * AuthRequest as returned by AuthRequestByID or AuthRequestByCode (above)
//
// * *oidc.JWTTokenRequest from a JWT that is the assertion value of a JWT Profile
//   Grant: https://datatracker.ietf.org/doc/html/rfc7523#section-2.1
func (s LocalAuthStorage) CreateAccessToken(_ context.Context, _ op.TokenRequest) (accessTokenID string, expiration time.Time, err error) {
	panic("not implemented") // TODO: Implement
}

// The TokenRequest parameter of CreateAccessAndRefreshTokens can be any of:
//
// * TokenRequest as returned by ClientCredentialsStorage.ClientCredentialsTokenRequest
//
// * RefreshTokenRequest as returned by AuthStorage.TokenRequestByRefreshToken
//
// * AuthRequest as by returned by the AuthRequestByID or AuthRequestByCode (above).
//   Used for the authorization code flow which requested offline_access scope and
//   registered the refresh_token grant type in advance
func (s LocalAuthStorage) CreateAccessAndRefreshTokens(
	ctx context.Context, request op.TokenRequest, currentRefreshToken string,
) (accessTokenID string, newRefreshTokenID string, expiration time.Time, err error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalAuthStorage) TokenRequestByRefreshToken(ctx context.Context, refreshTokenID string) (op.RefreshTokenRequest, error) {
	panic("not implemented") // TODO: Implement
}

func (s LocalAuthStorage) TerminateSession(ctx context.Context, userID string, clientID string) error {
	return nil
}

func (s LocalAuthStorage) RevokeToken(ctx context.Context, tokenID string, userID string, clientID string) *oidc.Error {
	return nil
}

func (s LocalAuthStorage) GetSigningKey(ctx context.Context, signkey chan<- jose.SigningKey) {
	signkey <- s.signkey
}

func (s LocalAuthStorage) GetKeySet(ctx context.Context) (*jose.JSONWebKeySet, error) {
	return s.jwks, nil
}
