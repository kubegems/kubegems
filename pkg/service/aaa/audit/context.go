package audit

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/models"
)

const (
	AuditDataKey      = "auditdata"
	AuditExtraDataKey = "auditextradata"
	AuditEnable       = "auditenable"
)

type AuditActionData struct {
	Action string
	Module string
	Name   string
}

// SetAuditData 设置上下文的审计数据
func (audit *DefaultAuditInstance) SetAuditData(c *gin.Context, action, mod, name string) {
	data := &AuditActionData{
		Action: action,
		Module: mod,
		Name:   name,
	}
	c.Set(AuditDataKey, data)
}

// SetExtraAuditData 设置上下文的审计数据 的系统环境信息（租户，项目，环境）
func (audit *DefaultAuditInstance) SetExtraAuditData(c *gin.Context, kind string, uid uint) {
	cacheTree := audit.cache.GetGlobalResourceTree()
	extra := cacheTree.Tree.FindParents(kind, uid)
	if len(extra) > 0 {
		c.Set(AuditExtraDataKey, extra)
	}
}

// SetExtraAuditDataByClusterNamespace 根据集群namesapce设置上下文的审计数据 的系统环境信息（租户，项目，环境）
func (audit *DefaultAuditInstance) SetExtraAuditDataByClusterNamespace(c *gin.Context, cluster, namesapce string) {
	cacheTree := audit.cache.GetGlobalResourceTree()
	node := cacheTree.Tree.FindNodeByClusterNamespace(cluster, namesapce)
	if node != nil {
		extra := cacheTree.Tree.FindParents(node.Kind, node.ID)
		if len(extra) > 0 {
			c.Set(AuditExtraDataKey, extra)
		}
	}
}

func (audit *DefaultAuditInstance) SetProxyAuditData(c *gin.Context, pobj *ProxyObject) {
}

func GetExtraAuditData(c *gin.Context) (string, map[string]string) {
	var tenant string
	tags := map[string]string{}
	extraIfe, exist := c.Get(AuditExtraDataKey)
	if !exist {
		return "", tags
	}
	extra := extraIfe.(models.ResourceQueue)
	for _, p := range extra {
		switch p.Kind {
		case models.ResTenant:
			tags["租户"] = p.Name
			tenant = p.Name
		case models.ResProject:
			tags["项目"] = p.Name
		case models.ResEnvironment:
			tags["环境"] = p.Name
			tags["集群"] = p.Cluster
			tags["namespace"] = p.Namespace
		}
	}
	return tenant, tags
}

func GetAuditData(c *gin.Context) *AuditActionData {
	dataIfe, exsit := c.Get(AuditDataKey)
	if !exsit {
		return nil
	}
	data := dataIfe.(*AuditActionData)
	return data
}

func GetProxyAuditData(c *gin.Context) (*AuditActionData, map[string]string) {
	objIfe, exist := c.Get("proxyobj")
	if !exist {
		return nil, nil
	}
	pobj := objIfe.(*ProxyObject)
	tags := map[string]string{
		"Cluster":   pobj.Cluster,
		"Namesapce": pobj.Namespace,
	}

	return &AuditActionData{
		Action: pobj.Action,
		Module: pobj.Version + "/" + pobj.Resource,
		Name:   pobj.Name,
	}, tags
}

func (audit *DefaultAuditInstance) SaveAuditLog(c *gin.Context) {
	// 所有GET请求不审计
	if c.Request.Method == http.MethodGet {
		return
	}
	bodyWriter := &responseBodyWriter{responseBody: bytes.NewBufferString(""), ResponseWriter: c.Writer}
	c.Writer = bodyWriter

	// 拷贝一份request 到reqBuf
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
		module = data.Module
		action = data.Action
		name = data.Name
	}

	tenant, tags = GetExtraAuditData(c)
	issuccess := c.Writer.Status() < http.StatusBadRequest

	audit.Log(user.GetUsername(), module, tenant, action, name, tags, raw, issuccess, c.ClientIP())
}
