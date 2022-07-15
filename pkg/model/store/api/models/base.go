package models

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/auth"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

func (o *ModelsAPI) IfPermission(req *restful.Request, resp *restful.Response, permission string, f func(ctx context.Context) (interface{}, error)) {
	info, _ := req.Attribute("user").(auth.UserInfo)
	if !o.authorization.HasPermission(req.Request.Context(), info.Username, permission) {
		response.Error(resp, response.StatusError{
			Status:  http.StatusForbidden,
			Message: fmt.Sprintf("user %s does not have permission %s", info.Username, permission),
		})
		return
	}

	if data, err := f(req.Request.Context()); err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, data)
	}
}

func (o *ModelsAPI) AddSourceAdmin(req *restful.Request, resp *restful.Response) {
	username := req.PathParameter("username")
	// permission = <resource>:<action>:<id>
	permission := fmt.Sprintf("source:*:%s", req.PathParameter("source"))

	if err := o.authorization.AddPermission(req.Request.Context(), username, permission); err != nil {
		response.Error(resp, err)
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
		response.Error(resp, err)
	} else {
		response.OK(resp, nil)
	}
}
