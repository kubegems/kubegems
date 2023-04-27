package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type PermissionChecker interface {
	// example:
	// HasPermission("alice", PermissionFromMethodPath("GET","/regions/region1/tenants/tenant1"))
	// HasPermission("alice", "regions:region1:tenants:tenant1:read")

	// HasPermission("alice", PermissionFromMethodPath("GET","/regions"))
	// HasPermission("alice", "regions:read")
	HasPermission(subject string, perm string) (bool, error)
}

type SimpleOperation interface {
	Add(name string, values ...string) error
	Remove(name string, values ...string) error
	Set(name string, values ...string) error
	Get(name string) []string
	List() map[string][]string
}

type AuthorizationManager interface {
	Roles() SimpleOperation
	RoleAuthorities() SimpleOperation
	UserRoles() SimpleOperation
}

type PermissionAction string

const (
	// It is recommended to use the ActionRead and ActionWrite constants when granting permissions.
	ActionRead  PermissionAction = "get,list,watch"                      // read is a combination of get, list and watch
	ActionWrite PermissionAction = "get,list,watch,create,update,remove" // if you have write, you have read as well
	// The following constants are provided for convenience.
	ActionCreate      PermissionAction = "create"
	ActionUpdate      PermissionAction = "update"
	ActionPatch       PermissionAction = "patch"
	ActionRemove      PermissionAction = "remove"
	ActionRemoveBatch PermissionAction = "removeBatch"
	ActionList        PermissionAction = "list"
	ActionGet         PermissionAction = "get"
	ActionWatch       PermissionAction = "watch"
	ActionUnknown     PermissionAction = ""
)

func Permission(action PermissionAction, target ...string) string {
	return strings.Join(append(target, string(action)), ":")
}

// plural
var MethodActionMapPlural = map[string]PermissionAction{
	"GET":    ActionList,
	"POST":   ActionCreate,
	"DELETE": ActionRemoveBatch,
}

// singular plural
var MethodActionMapSingular = map[string]PermissionAction{
	"GET":    ActionGet,
	"PUT":    ActionUpdate,
	"DELETE": ActionRemove,
	"PATCH":  ActionPatch,
}

func PermissionFromMethodPath(method string, path string) string {
	return Permission(MethodActionMapSingular[method], strings.Split(path, "/")[1:]...)
}

var _ PermissionChecker = &CasbinPermissionChecker{}

type CasbinPermissionChecker struct {
	enforcer *casbin.Enforcer
}

func NewCasbinPermissionChecker(ctx context.Context, db *gorm.DB) (*CasbinPermissionChecker, error) {
	casmodel, err := model.NewModelFromString(`
	[request_definition]
	r = sub, perm
	
	[policy_definition]
	p = sub, perm
	
	[role_definition]
	g = _, _

	[policy_effect]
	e = some(where (p.eft == allow))
	
	[matchers]
	m = g(r.sub, p.sub) && wildcardMatch(r.perm, p.perm)
	`)
	if err != nil {
		return nil, err
	}
	casadapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer(casmodel, casadapter)
	if err != nil {
		return nil, err
	}
	enforcer.AddFunction("wildcardMatch", WildcardMatchFunc)
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}
	return &CasbinPermissionChecker{enforcer: enforcer}, nil
}

func (c *CasbinPermissionChecker) HasPermission(subject string, perm string) (bool, error) {
	return c.enforcer.Enforce(subject, perm)
}

func WildcardMatchFunc(args ...interface{}) (interface{}, error) {
	// nolint: gomnd
	if len(args) < 2 {
		return false, nil
	}
	r, ok1 := args[0].(string)
	p, ok2 := args[1].(string)
	if !ok1 || !ok2 {
		return false, nil
	}
	return WildcardMatchSections(p, r), nil
}

func WildcardMatch(key1 string, key2 string) bool {
	return WildcardMatchSections(key2, key1)
}

type PathRewriteMatcher func(string) (string, bool)

func PrefixedPathRewriteMatcher(prefix string) PathRewriteMatcher {
	return func(path string) (string, bool) {
		return strings.TrimPrefix(path, prefix), true
	}
}

type SubjectExtractor func(r *http.Request) string

func DefaultSubjectExtractor() SubjectExtractor {
	return func(r *http.Request) string {
		return UsernameFromContext(r.Context())
	}
}

func NewPermissionCheckerHandler(matcher PathRewriteMatcher, extractor SubjectExtractor, authz PermissionChecker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path := r.Method, r.URL.Path
		if matcher != nil {
			if newPath, ok := matcher(path); ok {
				path = newPath
			} else {
				next.ServeHTTP(w, r)
				return
			}
		}
		subject := AnonymousUser
		if extractor != nil {
			subject = extractor(r)
		}
		ok, err := authz.HasPermission(subject, PermissionFromMethodPath(method, path))
		if err != nil {
			response.InternalServerError(w, err)
			return
		}
		if !ok {
			response.Forbidden(w, "access denied")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewPermissionCheckerMiddleware(matcher PathRewriteMatcher, extractor SubjectExtractor, authz PermissionChecker) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return NewPermissionCheckerHandler(matcher, extractor, authz, next)
	}
}
