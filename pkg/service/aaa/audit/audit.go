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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/aaa"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/models/cache"
	"kubegems.io/kubegems/pkg/utils/slice"
)

var normalActions = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
}

var methodMap = map[string]string{
	http.MethodDelete: i18n.Sprintf(context.TODO(), "delete"),
	http.MethodPatch:  i18n.Sprintf(context.TODO(), "patch"),
	http.MethodPut:    i18n.Sprintf(context.TODO(), "put"),
	http.MethodPost:   i18n.Sprintf(context.TODO(), "create"),
}

const (
	AuditMark   = "needAudit"
	AuditAction = "auditAction"
)

type AuditInterface interface {
	AuditProxyFunc(c *gin.Context, p *ProxyObject)
	WebsocketAuditFunc(username string, parents []cache.CommonResourceIface, ip string, proxyobj *ProxyObject) func(cmd string)

	SetAuditData(c *gin.Context, action, mod, name string)
	SetExtraAuditData(c *gin.Context, kind string, uid uint)
	SetExtraAuditDataByClusterNamespace(c *gin.Context, cluster, namesapce string)
}

type ProxyObject struct {
	NamespacedScoped bool
	Cluster          string
	Namespace        string
	Name             string
	Group            string
	Version          string
	Resource         string
	Action           string
}

func (p *ProxyObject) InNamespace() bool {
	if !p.NamespacedScoped {
		return false
	}
	return p.Namespace != "" && p.Namespace != "_" && p.Namespace != "_all"
}

type DefaultAuditInstance struct {
	userinterface aaa.ContextUserOperator
	cache         cache.ModelCache
	db            *gorm.DB
	logQueue      chan models.AuditLog
	logQueueClose bool
}

func NewAuditMiddleware(db *gorm.DB, cache cache.ModelCache, uinterface aaa.ContextUserOperator) *DefaultAuditInstance {
	audit := &DefaultAuditInstance{
		db:            db,
		logQueue:      make(chan models.AuditLog, 1000),
		cache:         cache,
		userinterface: uinterface,
	}
	return audit
}

func (audit *DefaultAuditInstance) AuditProxyFunc(c *gin.Context, proxyobj *ProxyObject) {
	if slice.ContainStr(normalActions, c.Request.Method) {
		return
	}
	action := methodMap[c.Request.Method]
	module := proxyobj.Resource
	name := proxyobj.Name
	audit.SetAuditData(c, action, module, name)
	if !proxyobj.InNamespace() {
		return
	}
	audit.SetExtraAuditDataByClusterNamespace(c, module, proxyobj.Namespace)
}

func (audit *DefaultAuditInstance) WebsocketAuditFunc(username string, parents []cache.CommonResourceIface, ip string, proxyobj *ProxyObject) func(cmd string) {
	var tenant string
	tags := map[string]string{}
	for _, p := range parents {
		switch p.GetKind() {
		case models.ResTenant:
			tenant = p.GetName()
			tags["tenant"] = p.GetName()
		case models.ResProject:
			tags["project"] = p.GetName()
		case models.ResEnvironment:
			tags["environment"] = p.GetName()
			tags["cluster"] = p.GetCluster()
			tags["namespace"] = p.GetNamespace()
		}
	}
	module := proxyobj.Name
	operation := proxyobj.Action
	return func(cmd string) {
		audit.Log(username, i18n.Sprintf(context.TODO(), "execute command"), tenant, operation, module, tags, cmd, true, ip)
	}
}

func (audit *DefaultAuditInstance) Log(username, module, tenant, operation, name string, labels map[string]string, raw interface{}, success bool, ip string) {
	rawjson, err := json.Marshal(raw)
	if err != nil {
		log.Errorf("can't record audit log: (%v)", raw)
	}

	labeljson, err := json.Marshal(labels)
	if err != nil {
		log.Errorf("can't record audit log: (%v)", raw)
	}
	auditLog := models.AuditLog{
		Username: username,
		Tenant:   tenant,
		Module:   module,
		Action:   operation,
		Name:     name,
		RawData:  rawjson,
		Labels:   labeljson,
		Success:  success,
		ClientIP: ip,
	}

	if audit.logQueueClose {
		return
	}

	audit.logQueue <- auditLog
}

func (audit *DefaultAuditInstance) Consumer(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			if len(audit.logQueue) == 0 {
				log.Info("audit log consumer exit")
				return nil
			}

			// stop receive new audit log
			close(audit.logQueue)
			audit.logQueueClose = true
			// consume all audit log in the queue
			wg := sync.WaitGroup{}
			for auditLog := range audit.logQueue {
				wg.Add(1)
				go func(auditLog models.AuditLog) {
					defer wg.Done()
					audit.emit(auditLog)
				}(auditLog)
			}
			wg.Wait()
			log.Info("audit log consumer all done, exit.")
			return nil
		case auditLog := <-audit.logQueue:
			go audit.emit(auditLog)
		}
	}
}

func (audit *DefaultAuditInstance) emit(auditLog models.AuditLog) {
	if err := audit.db.Create(&auditLog).Error; err != nil {
		o, _ := json.Marshal(auditLog)
		log.Errorf("can't record audit log: (%s), err: %v", string(o), err)
	}
}

func (audit *DefaultAuditInstance) Middleware() func(c *gin.Context) {
	return audit.SaveAuditLog
}

type requestBodyReader struct {
	rc io.ReadCloser
	w  io.Writer
}

func (reqReader *requestBodyReader) Read(p []byte) (n int, err error) {
	n, err = reqReader.rc.Read(p)
	if n > 0 {
		if n, err := reqReader.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return n, err
}

func (rc *requestBodyReader) Close() error {
	return rc.rc.Close()
}

type responseBodyWriter struct {
	gin.ResponseWriter
	responseBody *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.responseBody.Write(b)
	return r.ResponseWriter.Write(b)
}
