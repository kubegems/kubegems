package auth

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const MB = 1 << 20

type AuditOptions struct {
	RecordRead             bool     // Record read actions
	RecordBodyContentTypes []string // Record only for these content types
	MaxBodySize            int      // Max body size to record,0 means disable
}

func NewDefaultAuditOptions() *AuditOptions {
	return &AuditOptions{
		RecordRead: false,
		RecordBodyContentTypes: []string{
			"application/json",
			"application/xml",
			"application/x-www-form-urlencoded",
		},
		MaxBodySize: 1 * MB,
	}
}

type AuditRequest struct {
	HttpVersion string            `json:"httpVersion"` // http version
	Method      string            `json:"method"`      // method
	URL         string            `json:"url"`         // full url
	Header      map[string]string `json:"header"`      // header
	Body        []byte            `json:"body"`        // ignore body if size > 1MB or stream.
	ClientIP    string            `json:"clientIP"`    // client ip
}

type AuditResponse struct {
	StatusCode   int               `json:"statusCode"`   // status code
	Header       map[string]string `json:"header"`       // header
	ResponseBody []byte            `json:"responseBody"` // ignore body if size > 1MB or stream.
}

type AuditExtraMetadata map[string]string

type AuditLog struct {
	// request
	Request  AuditRequest  `json:"request"`
	Response AuditResponse `json:"response"`
	// authz
	Subject string `json:"subject"` // username
	// Resource is the resource type, e.g. "pods", "namespaces/default/pods/nginx-xxx"
	// we can detect the resource type and name from the request path.
	// GET  /zoos/{zoo_id}/animals/{animal_id} 	-> get zoos,zoo_id,animals,animal_id
	// GET  /zoos/{zoo_id}/animals 				-> list zoos,zoo_id,animals,animal_id
	// POST /zoos/{zoo_id}/animals:set-free 	-> set-free zoos,zoo_id,animals
	Action   string           `json:"action"`   // create, update, delete, get, list, set-free, etc.
	Domain   string           `json:"domain"`   // for multi-tenant
	Parents  []ParentResource `json:"parents"`  // parent resources, e.g. "zoos/{zoo_id}",
	Resource string           `json:"resource"` // resource type, e.g. "animals"
	Name     string           `json:"name"`     //  "{animal_id}", or "" if list
	// metadata
	StartTime time.Time          `json:"startTime"` // request start time
	EndTime   time.Time          `json:"endTime"`   // request end time
	Metadata  AuditExtraMetadata `json:"metadata"`  // extra metadata
}

type ParentResource struct {
	Resource string `json:"resource,omitempty"`
	Name     string `json:"name,omitempty"`
}

type auditmetadakeytype struct{}

var auditmetadakey = &auditmetadakeytype{}

func SetAuditExtraMeatadata(req *http.Request, k, v string) {
	extra, ok := req.Context().Value(auditmetadakey).(AuditExtraMetadata)
	if !ok {
		*req = *req.WithContext(context.WithValue(req.Context(), auditmetadakey, extra))
	}
	extra[k] = v
}

func GetAuditExtraMeatadata(req *http.Request) AuditExtraMetadata {
	extra, ok := req.Context().Value(auditmetadakey).(AuditExtraMetadata)
	if !ok {
		return nil
	}
	return extra
}

// Auditor is the interface to audit http request and response.
// Auditor must completes the audit log on request and response stage.
type HTTPAuditor interface {
	// Request is called on request stage, it returns the audit log and a wrapped response writer (if needed)
	OnRequest(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *AuditLog)
	// OnResponse is called on response stage, it passes the audit log and response writer produced by OnRequest
	OnResponse(w http.ResponseWriter, r *http.Request, auditlog *AuditLog)
}

type AuditSink interface {
	Save(log *AuditLog) error
}

type SimpleAuditor struct {
	Prefix  string // api prefix, e.g. /api/v1
	Options *AuditOptions
}

func NewSimpleAuditor(apiprefix string, options *AuditOptions, whitelist ...string) *SimpleAuditor {
	if !strings.HasPrefix(apiprefix, "/") {
		apiprefix = "/" + apiprefix
	}
	return &SimpleAuditor{
		Prefix:  apiprefix,
		Options: options,
	}
}

func (a *SimpleAuditor) OnRequest(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *AuditLog) {
	// not record read
	if !a.Options.RecordRead && r.Method == http.MethodGet {
		return w, nil
	}
	reqpath := r.URL.Path
	// not api request
	if !strings.HasPrefix(reqpath, a.Prefix) {
		return w, nil
	}
	reqpath = strings.TrimPrefix(reqpath, a.Prefix)

	auditlog := &AuditLog{
		Request: AuditRequest{
			HttpVersion: r.Proto,
			Method:      r.Method,
			URL:         r.URL.String(),
			Header:      HttpHeaderToMap(r.Header),
			Body:        ReadBodySafely(r, a.Options.RecordBodyContentTypes, a.Options.MaxBodySize),
			ClientIP:    ExtractClientIP(r),
		},
		StartTime: time.Now(),
	}
	a.CompleteAuditResource(r.Method, reqpath, auditlog)
	return NewStatusResponseWriter(w, a.Options.MaxBodySize), auditlog
}

func (a *SimpleAuditor) OnResponse(w http.ResponseWriter, r *http.Request, auditlog *AuditLog) {
	if auditlog == nil {
		return
	}
	auditlog.Metadata = GetAuditExtraMeatadata(r)
	auditlog.Subject = UsernameFromContext(r.Context())
	auditlog.EndTime = time.Now()
	auditlog.Response = AuditResponse{
		Header: HttpHeaderToMap(w.Header()),
	}
	if statusWriter, ok := w.(*StatusResponseWriter); ok {
		auditlog.Response.StatusCode = statusWriter.Code
		auditlog.Response.ResponseBody = statusWriter.Cache
	}
}

func ExtractClientIP(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-Ip")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	return clientIP
}

func (a *SimpleAuditor) CompleteAuditResource(method string, path string, auditlog *AuditLog) {
	// example:
	// /api/v1/namespaces/default/pods/nginx-xxx -> ["namespaces", "default", "pods", "nginx-xxx"]
	// /api/v1/namespaces/default/pods -> ["namespaces", "default", "pods"]
	// /api/v1/namespaces/default -> ["namespaces", "default"]
	// /api/v1/namespaces -> ["namespaces"]
	// /api/v1 -> []
	resource, action := splitResourceAction(path)
	parts := removeEmpty(strings.Split(resource, "/"))
	if len(parts) == 0 {
		return
	}
	// if odd, it's a list request, e.g. GET /api/v1/namespaces/default/pods
	if len(parts)%2 != 0 {
		parts = append(parts, "")
		if action == "" {
			action = string(MethodActionMapPlural[method])
		}
	} else {
		if action == "" {
			action = string(MethodActionMapSingular[method])
		}
	}
	parents, resource, name := []ParentResource{}, parts[0], parts[1]
	for i := 0; i < len(parts)/2-1; i++ {
		parents = append(parents, ParentResource{parts[i*2], parts[i*2+1]})
	}
	auditlog.Action, auditlog.Parents, auditlog.Resource, auditlog.Name = action, parents, resource, name
}

// e.g. /zoos/{id}/animals/{name}:feed -> /zoos/{id}/animals/{name},feed
func splitResourceAction(path string) (string, string) {
	if i := strings.LastIndex(path, ":"); i < 0 {
		return path, ""
	} else {
		return path[:i], path[i+1:]
	}
}

func NewHTTPAuditorHandler(auditor HTTPAuditor, sink AuditSink, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww, auditlog := auditor.OnRequest(w, r) // audit request
		if auditlog == nil {                    // no audit log, skip
			next.ServeHTTP(ww, r)
			return
		}
		if ww == nil {
			ww = w
		}
		next.ServeHTTP(ww, r)               // process request and response
		auditor.OnResponse(ww, r, auditlog) // audit response
		_ = sink.Save(auditlog)             // save audit log
	})
}

func NewHTTPAuditorMiddleware(auditor HTTPAuditor, sink AuditSink) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return NewHTTPAuditorHandler(auditor, sink, next)
	}
}
