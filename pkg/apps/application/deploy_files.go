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

package application

import (
	"github.com/gin-gonic/gin"
)

//	@Tags			Application
//	@Summary		列举文件
//	@Description	应用编排内容
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id		path		int											true	"tenaut id"
//	@Param			project_id		path		int											true	"project id"
//	@Param			environment_id	path		int											true	"environment_id"
//	@Param			name			path		string										true	"application name"
//	@Success		200				{object}	handlers.ResponseStruct{Data=[]FileContent}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/application/{name}/files [get]
//	@Security		JWT
func (h *ApplicationHandler) ListFiles(c *gin.Context) {
	h.Manifest.ListFiles(c)
}

//	@Tags			Application
//	@Summary		写入文件
//	@Description	修改应用编排
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id		path		int										true	"tenaut id"
//	@Param			project_id		path		int										true	"project id"
//	@Param			environment_id	path		int										true	"environment_id"
//	@Param			name			path		string									true	"application name"
//	@Param			filename		path		string									true	"file name"
//	@Param			body			body		string									true	"filecontent"
//	@Success		200				{object}	handlers.ResponseStruct{Data=string}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/files/{filename} [put]
//	@Security		JWT
func (h *ApplicationHandler) PutFile(c *gin.Context) {
	h.Manifest.PutFile(c)
}

//	@Tags			Application
//	@Summary		写入多个文件
//	@Description	修改应用编排
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id	path		int										true	"tenaut id"
//	@Param			project_id	path		int										true	"project id"
//	@Param			name		path		string									true	"name"
//	@Param			filename	path		string									true	"file name"
//	@Param			msg			query		string									true	"commit mesage"
//	@Param			body		body		[]FileContent							true	"files"
//	@Success		200			{object}	handlers.ResponseStruct{Data=string}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/files [put]
//	@Security		JWT
func (h *ApplicationHandler) PutFiles(c *gin.Context) {
	h.Manifest.PutFiles(c)
}

//	@Tags			Application
//	@Summary		删除文件
//	@Description	修改应用编排
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id		path		int										true	"tenaut id"
//	@Param			project_id		path		int										true	"project id"
//	@Param			environment_id	path		int										true	"environment_id"
//	@Param			name			path		string									true	"application name"
//	@Param			filename		path		string									true	"file name"
//	@Success		200				{object}	handlers.ResponseStruct{Data=string}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/files/{filename} [delete]
//	@Security		JWT
func (h *ApplicationHandler) RemoveFile(c *gin.Context) {
	h.Manifest.RemoveFile(c)
}

//	@Tags			Application
//	@Summary		应用编排文件历史
//	@Description	应用编排文件历史
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id		path		int											true	"tenaut id"
//	@Param			project_id		path		int											true	"project id"
//	@Param			environment_id	path		int											true	"environment_id"
//	@Param			name			path		string										true	"application name"
//	@Success		200				{object}	handlers.ResponseStruct{Data=[]FileContent}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitlog [get]
//	@Security		JWT
func (h *ApplicationHandler) GitLog(c *gin.Context) {
	h.Manifest.GitLog(c)
}

//	@Tags			Application
//	@Summary		应用编排文件diff
//	@Description	应用编排文件diff
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id		path		int										true	"tenaut id"
//	@Param			project_id		path		int										true	"project id"
//	@Param			environment_id	path		int										true	"environment_id"
//	@Param			name			path		string									true	"application name"
//	@Param			hash			query		string									true	"gitcommit hash"
//	@Success		200				{object}	handlers.ResponseStruct{Data=string}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitdiff [get]
//	@Security		JWT
func (h *ApplicationHandler) GitDiff(c *gin.Context) {
	h.Manifest.GitDiff(c)
}

//	@Tags			Application
//	@Summary		应用编排文件回滚
//	@Description	应用编排文件回滚
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id		path		int										true	"tenaut id"
//	@Param			project_id		path		int										true	"project id"
//	@Param			environment_id	path		int										true	"environment_id"
//	@Param			name			path		string									true	"application name"
//	@Param			hash			query		string									true	"gitcommit hash"
//	@Success		200				{object}	handlers.ResponseStruct{Data=string}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitrevert [post]
//	@Security		JWT
func (h *ApplicationHandler) GitRevert(c *gin.Context) {
	h.Manifest.GitRevert(c)
}

//	@Tags			Application
//	@Summary		应用编排文件刷新
//	@Description	应用编排文件刷新(git pull)
//	@Accept			json
//	@Produce		json
//	@Param			tenant_id		path		int										true	"tenaut id"
//	@Param			project_id		path		int										true	"project id"
//	@Param			environment_id	path		int										true	"environment_id"
//	@Param			name			path		string									true	"application name"
//	@Param			hash			query		string									true	"gitcommit hash"
//	@Success		200				{object}	handlers.ResponseStruct{Data=string}	"ok"
//	@Router			/v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitpull [post]
//	@Security		JWT
func (h *ApplicationHandler) GitPull(c *gin.Context) {
	h.Manifest.GitPull(c)
}
