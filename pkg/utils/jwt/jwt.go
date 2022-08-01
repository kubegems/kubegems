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

package jwt

import (
	"crypto/rsa"
	"io/ioutil"
	"time"

	"github.com/golang-jwt/jwt"
)

var jwtInstance *JWT

type JWT struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

type JWTClaims struct {
	*jwt.StandardClaims
	Payload interface{}
}

type Options struct {
	Expire time.Duration `yaml:"expire" default:"24h0m0s" help:"jwt expire time"`
	Cert   string        `yaml:"cert" default:"certs/jwt/tls.crt" help:"jwt cert file"`
	Key    string        `yaml:"key" default:"certs/jwt/tls.key" help:"jwt key file"`
}

func DefaultOptions() *Options {
	return &Options{
		Expire: time.Duration(time.Hour * 24),
		Cert:   "certs/jwt/tls.crt",
		Key:    "certs/jwt/tls.key",
	}
}

func (opts *Options) ToJWT() *JWT {
	if jwtInstance != nil {
		return jwtInstance
	}
	private, err := ioutil.ReadFile(opts.Key)
	if err != nil {
		panic(err)
	}
	public, err := ioutil.ReadFile(opts.Cert)
	if err != nil {
		panic(err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		panic(err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		panic(err)
	}
	jwtInstance = &JWT{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
	return jwtInstance
}

// GenerateToken Generate new jwt token
func (t *JWT) GenerateToken(payload interface{}, sub string, expire time.Duration) (token string, expriets int64, err error) {
	tk := jwt.New(jwt.GetSigningMethod("RS256"))
	now := time.Now()
	expriets = now.Add(expire).Unix()
	tk.Claims = wrapClaims(payload, sub, now, expriets)
	token, err = tk.SignedString(t.privateKey)
	return token, expriets, err
}

// ParseToken Parse jwt token, return the claims
func (t *JWT) ParseToken(token string) (*JWTClaims, error) {
	claims := JWTClaims{}
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return t.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if err := claims.Valid(); err != nil {
		return nil, err
	}
	return &claims, err
}

func wrapClaims(v interface{}, sub string, now time.Time, expirets int64) *JWTClaims {
	return &JWTClaims{
		Payload: v,
		StandardClaims: &jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: expirets,
			Subject:   sub,
		},
	}
}
