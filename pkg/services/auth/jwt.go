package auth

import (
	"crypto/rsa"
	"time"

	"github.com/golang-jwt/jwt"
)

type JWT struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

type JWTClaims struct {
	*jwt.StandardClaims
	Payload interface{}
}

type JWTOptions struct {
	Expire   time.Duration `yaml:"expire" default:"24h0m0s" help:"jwt expire time"`
	Cert     string        `yaml:"cert" default:"certs/jwt/tls.crt" help:"jwt cert file"`
	Key      string        `yaml:"key" default:"certs/jwt/tls.key" help:"jwt key file"`
	CertData string        `yaml:"cert_data" default:"" help:"jwt cert data"`
	KeyData  string        `yaml:"key_data" default:"" help:"jwt key data"`
}

// GenerateToken Generate new jwt token
func (t *JWT) GenerateToken(payload interface{}, expire time.Duration) (token string, expriets int64, err error) {
	tk := jwt.New(jwt.GetSigningMethod("RS256"))
	now := time.Now()
	expriets = now.Add(expire).Unix()
	tk.Claims = wrapClaims(payload, now, expriets)
	token, err = tk.SignedString(t.privateKey)
	return token, expriets, err
}

// ParseToken Parse jwt token, return the claims
func (t *JWT) ParseToken(token string) (*JWTClaims, error) {
	claims := JWTClaims{}
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return t.publicKey, nil
	})
	return &claims, err
}

func wrapClaims(v interface{}, now time.Time, expirets int64) *JWTClaims {
	return &JWTClaims{
		Payload: v,
		StandardClaims: &jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: expirets,
		},
	}
}
