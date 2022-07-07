package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

const (
	PermissionAdmin = "*:*:*"
	PermissionNone  = ""
)

func Permission(resource, action, id string) string {
	return fmt.Sprintf("%s:%s:%s", resource, action, id)
}

type AuthorizationManager interface {
	AddPermission(ctx context.Context, username string, permissions string) error
	ListPermissions(ctx context.Context, username string) ([]string, error)
	ListUsersHasPermission(ctx context.Context, permission string) ([]string, error)
	RemovePermission(ctx context.Context, username string, permissions string) error
	HasPermission(ctx context.Context, username string, permission string) bool
}

type LocalAuthorization struct {
	repository *repository.AuthorizationRepository
}

func NewLocalAuthorization(ctx context.Context, db *mongo.Database) *LocalAuthorization {
	return &LocalAuthorization{repository: repository.NewAuthorizationRepository(ctx, db)}
}

func (a *LocalAuthorization) Init(ctx context.Context) error {
	return a.repository.InitSchema(ctx)
}

func (a *LocalAuthorization) AddPermission(ctx context.Context, username string, permission string) error {
	authorization, err := a.repository.Get(ctx, username)
	if err != nil {
		return err
	}

	exist := false

	for _, p := range authorization.Permissions {
		if p == permission {
			exist = true
			break
		}
	}

	if exist {
		return nil
	}
	authorization.Permissions = append(authorization.Permissions, permission)
	return a.repository.Set(ctx, authorization)
}

func (a *LocalAuthorization) RemovePermission(ctx context.Context, username string, permission string) error {
	authorization, err := a.repository.Get(ctx, username)
	if err != nil {
		return err
	}
	for i, p := range authorization.Permissions {
		if p == permission {
			authorization.Permissions = append(authorization.Permissions[:i], authorization.Permissions[i+1:]...)
			return a.repository.Set(ctx, authorization)
		}
	}
	return nil
}

func (a *LocalAuthorization) ListPermissions(ctx context.Context, username string) ([]string, error) {
	authorization, err := a.repository.Get(ctx, username)
	if err != nil {
		return nil, err
	}
	return authorization.Permissions, nil
}

func (a *LocalAuthorization) ListUsersHasPermission(ctx context.Context, permissionRegexp string) ([]string, error) {
	list, err := a.repository.List(ctx, permissionRegexp)
	if err != nil {
		return nil, err
	}
	users := make([]string, 0, len(list))
	for _, auth := range list {
		users = append(users, auth.Username)
	}
	return users, nil
}

func (a *LocalAuthorization) HasPermission(ctx context.Context, username string, permission string) bool {
	return true

	permissions, err := a.ListPermissions(ctx, username)
	if err != nil {
		return false
	}
	for _, p := range permissions {
		// TODO: use wildcard
		if p == permission {
			return true
		}
	}
	return false
}

func (o *ModelsAPI) IfPermission(req *restful.Request, resp *restful.Response, permission string, f func(ctx context.Context) (interface{}, error)) {
	info, _ := req.Attribute("user").(UserInfo)
	if !o.authorization.HasPermission(req.Request.Context(), info.Username, permission) {
		response.ErrorResponse(resp, response.StatusError{
			Status:  http.StatusForbidden,
			Message: fmt.Sprintf("user %s does not have permission %s", info.Username, permission),
		})
		return
	}

	if data, err := f(req.Request.Context()); err != nil {
		response.ErrorResponse(resp, err)
	} else {
		response.OK(resp, data)
	}
}

func (o *ModelsAPI) AddSourceAdmin(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("username")
	// permission = <resource>:<action>:<id>
	permission := fmt.Sprintf("source:*:%s", req.PathParameter("source"))

	if err := o.authorization.AddPermission(req.Request.Context(), username, permission); err != nil {
		response.ErrorResponse(resp, err)
	} else {
		response.OK(resp, nil)
	}
}

func (o *ModelsAPI) ListSourceAdmin(req *restful.Request, resp *restful.Response) {
	// info, _ := req.Attribute("user").(UserInfo)
	permissionRegexp := fmt.Sprintf("source:\\*:%s", req.PathParameter("source"))
	users, err := o.authorization.ListUsersHasPermission(req.Request.Context(), permissionRegexp)
	if err != nil {
		response.ServerError(resp, err)
		return
	}
	response.OK(resp, users)
}

func (o *ModelsAPI) DeleteSourceAdmin(req *restful.Request, resp *restful.Response) {
	// info, _ := req.Attribute("user").(UserInfo)
	username := req.PathParameter("username")
	permission := fmt.Sprintf("source:*:%s", req.PathParameter("source"))

	ctx := req.Request.Context()
	if err := o.authorization.RemovePermission(ctx, username, permission); err != nil {
		response.ErrorResponse(resp, err)
	} else {
		response.OK(resp, nil)
	}
}
