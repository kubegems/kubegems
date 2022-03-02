package filters

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/auth/user"
	"kubegems.io/pkg/utils/jwt"
)

type AuthMiddleware struct {
	getters []UserGetterIface
}

func NewAuthMiddleware(opts *jwt.Options) *AuthMiddleware {
	var getters []UserGetterIface
	getters = append(getters, &BearerTokenUserLoader{
		JWT: opts.ToJWT(),
	})
	getters = append(getters, &PrivateTokenUserLoader{})
	return &AuthMiddleware{
		getters: getters,
	}
}

// UserLoader 根据凭据载入用户
func (l *AuthMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if isOpen(req) {
		chain.ProcessFilter(req, resp)
		return
	}
	if len(l.getters) > 0 {
		var (
			loaded bool
			user   user.CommonUserIface
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
}

// UserGetterIface 用户接口
type UserGetterIface interface {
	GetUser(req *http.Request) (u user.CommonUserIface, exist bool)
}

// BearerTokenUserLoader  bearer类型
type BearerTokenUserLoader struct {
	JWT *jwt.JWT
}

func (l *BearerTokenUserLoader) GetUser(req *http.Request) (u user.CommonUserIface, exist bool) {
	htype, token := parseAuthorizationHeader(req)
	if strings.ToLower(htype) != "bearer" {
		return nil, false
	}
	claims, err := l.JWT.ParseToken(token)
	if err != nil {
		log.Error(err, "flow", "parse jwt token")
		return nil, false
	}
	bts, _ := json.Marshal(claims.Payload)
	var user models.UserCommon
	err = json.Unmarshal(bts, &user)
	if err != nil {
		log.Error(err, "failed to load userinfo", "data", string(bts))
	}
	return &user, err == nil
}

// PrivateTokenUserLoader private-token认证
type PrivateTokenUserLoader struct{}

func (l *PrivateTokenUserLoader) GetUser(req *http.Request) (u user.CommonUserIface, exist bool) {
	ptoken := req.Header.Get("PRIVATE-TOKEN")
	fmt.Println(ptoken)
	// TODO: finish logic
	return nil, false
}

func parseAuthorizationHeader(req *http.Request) (htype, token string) {
	authheader := req.Header.Get("Authorization")
	if authheader == "" {
		tkn := req.URL.Query().Get("token")
		if tkn == "" {
			return
		}
		htype = "Bearer"
		token = tkn
		q := req.URL.Query()
		q.Del("token")
		req.URL.RawQuery = q.Encode()
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

func (l *BasicAuthUserLoader) GetUser(req *http.Request) (userData user.CommonUserIface, exist bool) {
	htype, token := parseAuthorizationHeader(req)
	if strings.ToLower(htype) != "basic" {
		return nil, false
	}
	bts, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		log.Error(err, "flow", "parse private token")
		return nil, false
	}
	seps := bytes.SplitN(bts, []byte(":"), 2)
	username := string(seps[0])
	password := string(seps[1])
	fmt.Println(username, password)
	// TODO: finish logic
	return nil, false
}
