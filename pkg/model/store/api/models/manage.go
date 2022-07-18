package models

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

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

func (m *ModelsAPI) registerSourceAdminRoute() *route.Group {
	return route.
		NewGroup("/admins").
		Tag("admins").
		AddRoutes(
			route.GET("").To(m.ListSourceAdmin).Doc("list admins").Response([]string{}),
			route.POST("/{username}").To(m.AddSourceAdmin).Doc("add source admin").
				Parameters(route.PathParameter("username", "Username of admin")).
				Accept("*/*"),
			route.DELETE("/{username}").To(m.DeleteSourceAdmin).Doc("delete source admin").Parameters(
				route.PathParameter("username", "Username of admin"),
			),
		)
}
