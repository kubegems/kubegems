package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/service/models/cache"
	"kubegems.io/pkg/utils/slice"
)

var normalActions = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
}

var methodMap = map[string]string{
	http.MethodDelete: "删除",
	http.MethodPatch:  "patch修改",
	http.MethodPut:    "put修改",
	http.MethodPost:   "创建",
}

const (
	AuditMark   = "needAudit"
	AuditAction = "auditAction"
)

type AuditInterface interface {
	AuditProxyFunc(c *gin.Context, p *ProxyObject)
	WebsocketAuditFunc(username string, parents []cache.CommonResourceIface, ip string, proxyobj *ProxyObject) func(cmd string)

	// 重构版本新加的方法
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
	cache         *cache.ModelCache
	db            *gorm.DB
	logQueue      chan models.AuditLog
}

func NewAuditMiddleware(db *gorm.DB, cache *cache.ModelCache, uinterface aaa.ContextUserOperator) *DefaultAuditInstance {
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
	n := audit.cache.FindEnvironment(proxyobj.Cluster, proxyobj.Namespace)
	if n != nil {
		extra := audit.cache.FindParents(n.GetKind(), n.GetID())
		if len(extra) > 0 {
			c.Set(AuditExtraDataKey, extra)
		}
	}
}

func (audit *DefaultAuditInstance) WebsocketAuditFunc(username string, parents []cache.CommonResourceIface, ip string, proxyobj *ProxyObject) func(cmd string) {
	var tenant string
	tags := map[string]string{}
	for _, p := range parents {
		switch p.GetKind() {
		case models.ResTenant:
			tenant = p.GetName()
			tags["租户"] = p.GetName()
		case models.ResProject:
			tags["项目"] = p.GetName()
		case models.ResEnvironment:
			tags["环境"] = p.GetName()
			tags["集群"] = p.GetCluster()
			tags["namespace"] = p.GetNamespace()
		}
	}
	module := proxyobj.Name
	operation := proxyobj.Action
	return func(cmd string) {
		audit.Log(username, "执行命令", tenant, operation, module, tags, cmd, true, ip)
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
	audit.logQueue <- auditLog
}

func (audit *DefaultAuditInstance) Consumer(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Info("audit log consumer exit")
			return nil
		case auditLog := <-audit.logQueue:
			if err := audit.db.Create(&auditLog).Error; err != nil {
				o, _ := json.Marshal(auditLog)
				log.Errorf("can't record audit log: (%s), err: %v", string(o), err)
			}
		}
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
