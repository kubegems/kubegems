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

package models

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/library/rest/response"
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
