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
