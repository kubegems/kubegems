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

package audit

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/models"
)

const (
	AuditSubjectKey   = "audit_subject"
	AuditExtraDataKey = "audit_extra_datas"
)

// SetAuditData  set audit data in context
func (audit *DefaultAuditInstance) SetAuditData(c *gin.Context, action, module, name string) {
	subject := map[string]string{
		"action": action,
		"module": module,
		"name":   name,
	}
	c.Set(AuditSubjectKey, subject)
}

// SetExtraAuditData set audit extra data in context
func (audit *DefaultAuditInstance) SetExtraAuditData(c *gin.Context, kind string, uid uint) {
	var ctxdata map[string]string
	contextDatas := audit.cache.FindParents(kind, uid)
	extra, exist := c.Get(AuditExtraDataKey)
	if exist {
		ctxdata = extra.(map[string]string)
	} else {
		ctxdata = make(map[string]string)
	}
	for _, cData := range contextDatas {
		switch cData.GetKind() {
		case models.ResTenant:
			ctxdata["tenant"] = cData.GetName()
		case models.ResProject:
			ctxdata["project"] = cData.GetName()
		case models.ResEnvironment:
			ctxdata["environment"] = cData.GetName()
			ctxdata["cluster"] = cData.GetCluster()
			ctxdata["namespace"] = cData.GetNamespace()
		}
	}
	c.Set(AuditExtraDataKey, ctxdata)

}

// SetExtraAuditDataByClusterNamespace  set context audit info via namespapce
func (audit *DefaultAuditInstance) SetExtraAuditDataByClusterNamespace(c *gin.Context, cluster, namespace string) {
	env := audit.cache.FindEnvironment(cluster, namespace)
	if env == nil {
		return
	}
	audit.SetExtraAuditData(c, models.ResEnvironment, env.GetID())
}

func GetExtraAuditData(c *gin.Context) (string, map[string]string) {
	var tenant string
	tags := map[string]string{}
	contextDataIface, exist := c.Get(AuditExtraDataKey)
	if !exist {
		return tenant, tags
	}
	contextDatas, ok := contextDataIface.(map[string]string)
	if !ok {
		return tenant, tags
	}
	if v, exist := contextDatas["tenant"]; exist {
		tenant = v
		tags["租户"] = tenant
	}
	if v, exist := contextDatas["project"]; exist {
		tags["项目"] = v
	}
	if v, exist := contextDatas["application"]; exist {
		tags["应用"] = v
	}
	if v, exist := contextDatas["environment"]; exist {
		tags["环境"] = v
	}
	if v, exist := contextDatas["cluster"]; exist {
		tags["集群"] = v
	}
	if v, exist := contextDatas["namespace"]; exist {
		tags["namespace"] = v
	}
	return tenant, tags
}

func GetAuditData(c *gin.Context) map[string]string {
	subjectIface, exsit := c.Get(AuditSubjectKey)
	if !exsit {
		return nil
	}
	return subjectIface.(map[string]string)
}

func (audit *DefaultAuditInstance) SaveAuditLog(c *gin.Context) {
	// ingore GET methods, not audit
	if c.Request.Method == http.MethodGet {
		return
	}
	bodyWriter := &responseBodyWriter{responseBody: bytes.NewBufferString(""), ResponseWriter: c.Writer}
	c.Writer = bodyWriter

	reqBuf := bytes.NewBufferString("")
	newBody := &requestBodyReader{c.Request.Body, reqBuf}
	c.Request.Body = newBody

	t := time.Now()
	c.Next()
	delta := time.Since(t)
	rawdata := gin.H{
		"statusCode": c.Writer.Status(),
		"duration":   delta.String(),
		"rawuri":     c.Request.RequestURI,
		"request":    reqBuf.String(),
		"response":   bodyWriter.responseBody.String(),
	}
	audit.LogIt(c, t, rawdata)
}

func (audit *DefaultAuditInstance) LogIt(c *gin.Context, t time.Time, raw gin.H) {
	user, _ := audit.userinterface.GetContextUser(c)

	var (
		module string
		tenant string
		action string
		name   string
		tags   map[string]string
	)

	if data := GetAuditData(c); data != nil {
		module = data["module"]
		action = data["action"]
		name = data["name"]
	}

	tenant, tags = GetExtraAuditData(c)
	issuccess := c.Writer.Status() < http.StatusBadRequest

	audit.Log(user.GetUsername(), module, tenant, action, name, tags, raw, issuccess, c.ClientIP())
}
