package application

import "github.com/emicklei/go-restful/v3"

// @Tags         Application
// @Summary      列举文件
// @Description  应用编排内容
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @Param        environment_id  path      int                                          true  "environment_id"
// @Param        name            path      string                                       true  "application name"
// @Success      200             {object}  handlers.ResponseStruct{Data=[]FileContent}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/application/{name}/files [get]
// @Security     JWT
func (h *ApplicationHandler) ListFiles(req *restful.Request, resp *restful.Response) {
	h.Manifest.ListFiles(req, resp)
}

// @Tags         Application
// @Summary      写入文件
// @Description  修改应用编排
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @Param        environment_id  path      int                                   true  "environment_id"
// @Param        name            path      string                                true  "application name"
// @Param        filename        path      string                                true  "file name"
// @Param        body            body      string                                true  "filecontent"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/files/{filename} [put]
// @Security     JWT
func (h *ApplicationHandler) UpdateFile(req *restful.Request, resp *restful.Response) {
	h.Manifest.UpdateFile(req, resp)
}

// @Tags         Application
// @Summary      写入多个文件
// @Description  修改应用编排
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                   true  "tenaut id"
// @Param        project_id  path      int                                   true  "project id"
// @Param        name        path      string                                true  "name"
// @Param        filename    path      string                                true  "file name"
// @Param        msg         query     string                                true  "commit mesage"
// @Param        body        body      []FileContent                         true  "files"
// @Success      200         {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/files [put]
// @Security     JWT
func (h *ApplicationHandler) UpdateFiles(req *restful.Request, resp *restful.Response) {
	h.Manifest.UpdateFiles(req, resp)
}

// @Tags         Application
// @Summary      删除文件
// @Description  修改应用编排
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @Param        environment_id  path      int                                   true  "environment_id"
// @Param        name            path      string                                true  "application name"
// @Param        filename        path      string                                true  "file name"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/files/{filename} [delete]
// @Security     JWT
func (h *ApplicationHandler) DeleteFile(req *restful.Request, resp *restful.Response) {
	h.Manifest.DeleteFile(req, resp)
}

// @Tags         Application
// @Summary      应用编排文件历史
// @Description  应用编排文件历史
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @Param        environment_id  path      int                                          true  "environment_id"
// @Param        name            path      string                                       true  "application name"
// @Success      200             {object}  handlers.ResponseStruct{Data=[]FileContent}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitlog [get]
// @Security     JWT
func (h *ApplicationHandler) GitLog(req *restful.Request, resp *restful.Response) {
	h.Manifest.GitLog(req, resp)
}

// @Tags         Application
// @Summary      应用编排文件diff
// @Description  应用编排文件diff
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @Param        environment_id  path      int                                   true  "environment_id"
// @Param        name            path      string                                true  "application name"
// @Param        hash            query     string                                true  "gitcommit hash"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitdiff [get]
// @Security     JWT
func (h *ApplicationHandler) GitDiff(req *restful.Request, resp *restful.Response) {
	h.Manifest.GitDiff(req, resp)
}

// @Tags         Application
// @Summary      应用编排文件回滚
// @Description  应用编排文件回滚
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                          true  "tenaut id"
// @Param        project_id      path      int                                          true  "project id"
// @Param        environment_id  path      int                                   true  "environment_id"
// @Param        name            path      string                                true  "application name"
// @Param        hash            query     string                                true  "gitcommit hash"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitrevert [post]
// @Security     JWT
func (h *ApplicationHandler) GitRevert(req *restful.Request, resp *restful.Response) {
	h.Manifest.GitRevert(req, resp)
}

// @Tags         Application
// @Summary      应用编排文件刷新
// @Description  应用编排文件刷新(git pull)
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                          true  "tenaut id"
// @Param        project_id      path      int                                          true  "project id"
// @Param        environment_id  path      int                                   true  "environment_id"
// @Param        name            path      string                                true  "application name"
// @Param        hash            query     string                                true  "gitcommit hash"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/gitpull [post]
// @Security     JWT
func (h *ApplicationHandler) GitPull(req *restful.Request, resp *restful.Response) {
	h.Manifest.GitPull(req, resp)
}
