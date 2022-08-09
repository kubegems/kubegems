package oidc

import (
	"context"
	"crypto/sha256"
	"os"

	"github.com/emicklei/go-restful/v3"
	"github.com/zitadel/oidc/pkg/oidc"
	"github.com/zitadel/oidc/pkg/op"
	"golang.org/x/text/language"
	"kubegems.io/kubegems/pkg/utils/route"
)

const (
	pathLoggedOut = "/logged-out"
)

type OIDCProvider struct {
	OP op.OpenIDProvider
}

func NewProvider(ctx context.Context, options *OIDCOptions) (*OIDCProvider, error) {
	os.Setenv(op.OidcDevMode, "true") // to allow http issuer
	config := &op.Config{
		Issuer:    options.Issuer,
		CryptoKey: sha256.Sum256([]byte("kubegems")),
		// will be used if the end_session endpoint is called without a post_logout_redirect_uri
		DefaultLogoutRedirectURI: pathLoggedOut,
		// enables code_challenge_method S256 for PKCE (and therefore PKCE in general)
		CodeMethodS256: true,
		// enables additional client_id/client_secret authentication by form post (not only HTTP Basic Auth)
		AuthMethodPost: true,
		// enables additional authentication by using private_key_jwt
		AuthMethodPrivateKeyJWT: true,
		// enables refresh_token grant use
		GrantTypeRefreshToken: true,
		// enables use of the `request` Object parameter
		RequestObjectSupported: true,
		// this example has only static texts (in English), so we'll set the here accordingly
		SupportedUILocales: []language.Tag{language.English},
	}
	storage, err := NewLocalStorage(ctx, options)
	if err != nil {
		return nil, err
	}
	provider, err := op.NewOpenIDProvider(ctx, config, storage)
	if err != nil {
		return nil, err
	}
	return &OIDCProvider{OP: provider}, nil
}

func (m *OIDCProvider) RegisterRoute(rg *route.Group) {
	handler := m.OP.HttpHandler()
	wraphandler := func(req *restful.Request, resp *restful.Response) {
		handler.ServeHTTP(resp.ResponseWriter, req.Request)
	}
	rg.AddRoutes(
		route.GET(m.OP.KeysEndpoint().Relative()).To(wraphandler),
		route.GET(oidc.DiscoveryEndpoint).To(wraphandler),
	)
}
