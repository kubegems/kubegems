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

package api

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

const PluginRepositoriesName = "plugin-repositories"

func (o *PluginsAPI) RepoUpdate(req *restful.Request, resp *restful.Response) {
	reponame := req.PathParameter("name")
	if repo, err := o.PM.GetRepo(req.Request.Context(), reponame); err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, repo)
	}
}

func (o *PluginsAPI) RepoList(req *restful.Request, resp *restful.Response) {
	repos, err := o.PM.ListRepos(req.Request.Context())
	if err != nil {
		response.Error(resp, err)
		return
	}

	response.OK(resp, repos)
}

func (o *PluginsAPI) RepoGet(req *restful.Request, resp *restful.Response) {
	reponame := req.PathParameter("name")
	repos, err := o.PM.ListRepos(req.Request.Context())
	if err != nil {
		response.Error(resp, err)
		return
	}
	for _, repo := range repos {
		if repo.Name == reponame {
			response.OK(resp, repo)
			return
		}
	}
	response.NotFound(resp, fmt.Sprintf("repo %s not found", reponame))
}

func (o *PluginsAPI) RepoAdd(req *restful.Request, resp *restful.Response) {
	repo := &pluginmanager.Repository{}
	if err := request.Body(req.Request, &repo); err != nil {
		response.Error(resp, err)
		return
	}
	if err := o.PM.SetRepo(req.Request.Context(), repo, true); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, repo)
}

func (o *PluginsAPI) RepoRemove(req *restful.Request, resp *restful.Response) {
	reponame := req.PathParameter("name")
	if err := o.PM.DeleteRepo(req.Request.Context(), reponame); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, "ok")
}
