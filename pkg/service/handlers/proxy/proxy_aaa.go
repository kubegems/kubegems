package proxy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/service/aaa/audit"
)

func g(gn string) string {
	return fmt.Sprintf("(?P<%s>[a-zA-Z0-9._-]+?)", gn)
}

var (
	namespace = g("namespace")
	name      = g("name")
	group     = g("group")
	version   = g("version")
	resource  = g("resource")
	action    = g("action")

	regGVR     = regexp.MustCompile(fmt.Sprintf("^/%s/%s/%s$", group, version, resource))
	regGVRN    = regexp.MustCompile(fmt.Sprintf("^/%s/%s/%s/%s$", group, version, resource, name))
	regGVRNA   = regexp.MustCompile(fmt.Sprintf("^/%s/%s/%s/%s/actions/%s$", group, version, resource, name, action))
	regGVRNS   = regexp.MustCompile(fmt.Sprintf("^/%s/%s/namespaces/%s/%s$", group, version, namespace, resource))
	regGVRNSN  = regexp.MustCompile(fmt.Sprintf("^/%s/%s/namespaces/%s/%s/%s$", group, version, namespace, resource, name))
	regGVRNSNA = regexp.MustCompile(fmt.Sprintf("^/%s/%s/namespaces/%s/%s/%s/actions/%s$", group, version, namespace, resource, name, action))
	regs       = []*regexp.Regexp{regGVR, regGVRN, regGVRNA, regGVRNS, regGVRNSN, regGVRNSNA}
)

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

func ParseProxyObj(c *gin.Context, path string) *audit.ProxyObject {
	proxyobj := audit.ProxyObject{
		Cluster: c.Param("cluster"),
	}
	tpath := path
	if strings.HasPrefix(path, "/custom") {
		tpath = strings.TrimPrefix(path, "/custom")
	}
	parsePath(tpath, &proxyobj)
	return &proxyobj
}

func parsePath(path string, obj *audit.ProxyObject) {
	for _, reg := range regs {
		if reg.MatchString(path) {
			fillObjectFields(reg, obj, path)
			return
		}
	}
}

func fillObjectFields(r *regexp.Regexp, obj *audit.ProxyObject, path string) {
	ret := map[string]string{}
	names := r.SubexpNames()
	subs := r.FindStringSubmatch(path)
	for idx := range names {
		if idx != 0 {
			ret[names[idx]] = subs[idx]
		}
	}
	obj.Namespace = ret["namespace"]
	obj.Name = ret["name"]
	obj.Group = ret["group"]
	obj.Version = ret["version"]
	obj.Resource = ret["resource"]
	obj.Action = ret["action"]
	if len(obj.Namespace) > 0 {
		obj.NamespacedScoped = true
	}
}
