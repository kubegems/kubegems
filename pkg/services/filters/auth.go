package filters

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/services/auth"
)

type AuthMiddleware struct {
	getters []UserGetterIface
}

func NewAuthMiddleware() *AuthMiddleware {
	var getters []UserGetterIface
	getters = append(getters, &BearerTokenUserLoader{})
	getters = append(getters, &PrivateTokenUserLoader{})
	return &AuthMiddleware{
		getters: getters,
	}
}

// UserLoader 根据凭据载入用户
func (l *AuthMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	fmt.Println("TODO: before userloader")
	if len(l.getters) > 0 {
		var (
			loaded bool
			user   client.CommonUserIfe
		)
		for idx := range l.getters {
			user, loaded = l.getters[idx].GetUser(req.Request)
			if loaded {
				break
			}
		}
		if !loaded {
			resp.WriteHeaderAndJson(http.StatusUnauthorized, "unauthorized", restful.MIME_JSON)
			return
		}
		req.SetAttribute("user", user)
	}
	chain.ProcessFilter(req, resp)
	fmt.Println("TODO: after userloader")
}

// UserGetterIface 用户接口
type UserGetterIface interface {
	GetUser(req *http.Request) (user client.CommonUserIfe, exist bool)
}

// BearerTokenUserLoader  bearer类型
type BearerTokenUserLoader struct {
	JWT *auth.JWT
}

func (l *BearerTokenUserLoader) GetUser(req *http.Request) (user client.CommonUserIfe, exist bool) {
	htype, token := parseAuthorizationHeader(req)
	if strings.ToLower(htype) != "bearer" {
		return nil, false
	}
	claims, err := l.JWT.ParseToken(token)
	if err != nil {
		return nil, false
	}
	user, y := claims.Payload.(client.CommonUserIfe)
	return user, y
}

// PrivateTokenUserLoader private-token认证
type PrivateTokenUserLoader struct{}

func (l *PrivateTokenUserLoader) GetUser(req *http.Request) (user client.CommonUserIfe, exist bool) {
	ptoken := req.Header.Get("PRIVATE-TOKEN")
	fmt.Println(ptoken)
	// TODO: finish logic
	return nil, false
}

func parseAuthorizationHeader(req *http.Request) (htype, token string) {
	authheader := req.Header.Get("Authorization")
	if authheader == "" {
		return
	}
	seps := strings.Split(authheader, " ")
	if len(seps) != 2 {
		return
	}
	return seps[0], seps[1]
}

// BasicAuthUserLoader basic认证
// 例如 Authorization: Basic YWxhZGRpbjpvcGVuc2VzYW1l
type BasicAuthUserLoader struct{}

func (l *BasicAuthUserLoader) GetUser(req *http.Request) (user client.CommonUserIfe, exist bool) {
	htype, token := parseAuthorizationHeader(req)
	if strings.ToLower(htype) != "basic" {
		return nil, false
	}
	bts, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, false
	}
	seps := bytes.SplitN(bts, []byte(":"), 2)
	username := string(seps[0])
	password := string(seps[1])
	fmt.Println(username, password)
	// TODO: finish logic
	return nil, false
}
