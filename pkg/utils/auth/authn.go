package auth

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

const AnonymousUser = "" // anonymous username

type OIDCOptions struct {
	Issuer   string `json:"issuer" description:"oidc issuer url"`
	Insecure bool   `json:"insecure" description:"skip issuer and audience verification (optional)"`
	Audience string `json:"audience" description:"oidc resource server audience (optional)"`
}

// TokenVerify is an interface for verifying access tokens.
// The returned token claims.
type TokenVerify interface {
	Verify(ctx context.Context, token string) (TokenClaims, error)
}

type TokenVerifyFunc func(ctx context.Context, token string) (TokenClaims, error)

func (f TokenVerifyFunc) Verify(ctx context.Context, token string) (TokenClaims, error) {
	return f(ctx, token)
}

func NewOIDCTokenVerify(ctx context.Context, options *OIDCOptions) (TokenVerify, error) {
	issuer, insecure := strings.TrimSuffix(options.Issuer, "/"), options.Insecure
	audience := options.Audience
	if audience == "" {
		audience = os.Getenv("OIDC_AUDIENCE")
	}
	if insecure {
		ctx = oidc.InsecureIssuerURLContext(ctx, issuer)
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID:          audience,
		SkipClientIDCheck: insecure,
		SkipIssuerCheck:   insecure,
	})
	return TokenVerifyFunc(func(ctx context.Context, token string) (TokenClaims, error) {
		// Though verifier.Verify() is used to verify id token, it can also be used to verify access token.
		// https://datatracker.ietf.org/doc/html/rfc7519#section-4.1
		accessToken, err := verifier.Verify(ctx, token)
		if err != nil {
			return TokenClaims{}, err
		}
		accessTokenClaims := map[string]any{}
		accessToken.Claims(&accessTokenClaims)
		return accessTokenClaims, nil
	}), nil
}

type contextAccessTokenClaims struct{}

var contextAccesstokenClaims = contextAccessTokenClaims{}

func UsernameFromContext(ctx context.Context) string {
	return TokenClaimsFromContext(ctx).Subject()
}

func TokenClaimsFromContext(ctx context.Context) TokenClaims {
	if username, ok := ctx.Value(contextAccesstokenClaims).(TokenClaims); ok {
		return username
	}
	return TokenClaims{}
}

func NewTokenClaimsContext(ctx context.Context, username TokenClaims) context.Context {
	return context.WithValue(ctx, contextAccesstokenClaims, username)
}

type TokenClaims map[string]any

func (t TokenClaims) Subject() string {
	if val, ok := t.Get("sub").(string); ok {
		return val
	}
	return AnonymousUser
}

func (t TokenClaims) Get(key string) any {
	return t[key]
}

// NewTokenVerifyHandler returns a http.Handler that verifies access tokens in the Authorization header.
// in next handler, the username is stored in the context and can be retrieved by UsernameFromContext(r.Context()).
// We acting as a resource server, so we need to verify the access token from the client.
func NewTokenVerifyHandler(authc TokenVerify, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("access_token")
		}
		if token == "" {
			response.Error(w, response.NewStatusErrorMessage(http.StatusUnauthorized, "missing access token"))
			return
		}
		claims, err := authc.Verify(r.Context(), token)
		if err != nil {
			response.Error(w, response.NewStatusErrorf(http.StatusUnauthorized, "invalid access token: %w", err))
			return
		}
		r.WithContext(NewTokenClaimsContext(r.Context(), claims))
		next.ServeHTTP(w, r)
	})
}

func NewTokenVerifyMiddleware(authc TokenVerify) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return NewTokenVerifyHandler(authc, next)
	}
}
