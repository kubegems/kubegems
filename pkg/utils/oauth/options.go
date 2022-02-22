package oauth

import "time"

type JWTOptions struct {
	JwtExpire time.Duration `json:"jwtExpire,omitempty" description:"jwt expire time"`
	JWTCert   string        `json:"jwtCert,omitempty" description:"jwt cert file"`
	JWTKey    string        `json:"jwtKey,omitempty" description:"jwt key file"`
}

func NewDefaultJWTOptions() *JWTOptions {
	return &JWTOptions{
		JwtExpire: 24 * time.Hour, // 24小时
		JWTCert:   "certs/jwt/tls.crt",
		JWTKey:    "certs/jwt/tls.key",
	}
}
