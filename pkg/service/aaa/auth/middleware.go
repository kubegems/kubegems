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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/aaa"
	"kubegems.io/kubegems/pkg/service/aaa/auth/user"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/jwt"
)

var tracer = otel.Tracer("auth")

type AuthMiddleware struct {
	getters []UserGetterIface
	uif     aaa.ContextUserOperator
}

func NewAuthMiddleware(opts *jwt.Options, userif aaa.ContextUserOperator) *AuthMiddleware {
	var getters []UserGetterIface
	getters = append(getters, &BearerTokenUserLoader{
		JWT: opts.ToJWT(),
	})
	getters = append(getters, &PrivateTokenUserLoader{})
	return &AuthMiddleware{
		getters: getters,
		uif:     userif,
	}
}

func (l *AuthMiddleware) FilterFunc(c *gin.Context) {
	if len(l.getters) > 0 {
		var (
			loaded bool
			user   models.CommonUserIface
		)
		for idx := range l.getters {
			user, loaded = l.getters[idx].GetUser(c.Request)
			if loaded {
				break
			}
		}
		if !loaded {
			c.AbortWithStatusJSON(http.StatusUnauthorized, i18n.Sprintf(c, "please login first"))
			return
		}
		l.uif.SetContextUser(c, user)
	}
	c.Next()
}

func (l *AuthMiddleware) GoRestfulMiddleware(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if len(l.getters) > 0 {
		var (
			loaded bool
			user   models.CommonUserIface
		)
		for idx := range l.getters {
			user, loaded = l.getters[idx].GetUser(req.Request)
			if loaded {
				break
			}
		}
		if !loaded {
			resp.WriteErrorString(http.StatusUnauthorized, "")
			return
		}
		// To get username
		// req.Attribute("username").(string)
		req.SetAttribute("username", user.GetUsername())
	}
	chain.ProcessFilter(req, resp)
}

// UserGetterIface
type UserGetterIface interface {
	GetUser(req *http.Request) (u user.CommonUserIface, exist bool)
}

// BearerTokenUserLoader  bearer type
type BearerTokenUserLoader struct {
	JWT *jwt.JWT
}

func (l *BearerTokenUserLoader) GetUser(req *http.Request) (u user.CommonUserIface, exist bool) {
	htype, token := parseAuthorizationHeader(req)
	if strings.ToLower(htype) != "bearer" {
		return nil, false
	}
	_, span := tracer.Start(req.Context(), "GetUser", trace.WithAttributes(attribute.String("token", token)))
	defer span.End()
	claims, err := l.JWT.ParseToken(token)
	if err != nil {
		log.Error(err, "parse jwt token")
		return nil, false
	}
	bts, _ := json.Marshal(claims.Payload)
	var user models.User
	err = json.Unmarshal(bts, &user)
	if err != nil {
		log.Error(err, "failed to load userinfo", "data", string(bts))
	}
	span.SetAttributes(attribute.Int("user.id", int(user.ID)), attribute.String("user.name", user.Username))
	return &user, err == nil
}

// PrivateTokenUserLoader private-token
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
// eg: Authorization: Basic YWxhZGRpbjpvcGVuc2VzYW1l
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
