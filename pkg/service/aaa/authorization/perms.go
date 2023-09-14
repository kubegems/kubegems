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

package authorization

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/service/aaa"
	"kubegems.io/kubegems/pkg/service/aaa/audit"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/models/cache"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/slice"
)

var normalActions = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
}

// PermissionManager 权限判断工具,仅支持租户，项目，环境三级数据
type PermissionManager interface {
	// CheckByClusterNamespace 根据cluster和namespace，判断是否有关联环境的权限
	CheckByClusterNamespace(c *gin.Context)
	// CheckByEnvironmentID 判断是否有环境的操作权限
	CheckByEnvironmentID(c *gin.Context)
	// CheckByProjectID  判断是否有项目的操作权限
	CheckByProjectID(c *gin.Context)
	// CheckByTenantID  判断是否有租户操作权限
	CheckByTenantID(c *gin.Context)
	// CheckByVirtualSpaceID  判断是否有虚拟空间操作权限
	CheckByVirtualSpaceID(c *gin.Context)
	// CheckIsSysADMIN  判断是否是系统管理员
	CheckIsSysADMIN(c *gin.Context)
	// CheckIsATenantAdmin  判断是否是一个租户管理员
	CheckIsATenantAdmin(c *gin.Context)
	// CheckCanDeployEnvironment  判断是否有对应环境的部署权限
	CheckCanDeployEnvironment(c *gin.Context)
}

type DefaultPermissionManager struct {
	Cache  *cache.ModelCache
	Userif aaa.ContextUserOperator
}

func (defaultPermChecker *DefaultPermissionManager) HasEnvPerm(c *gin.Context, cluster, namespace string) (hasPerm bool, objname string, currentrole string) {
	user, exist := defaultPermChecker.Userif.GetContextUser(c)
	if !exist {
		return false, "", ""
	}
	userAuthoriy := defaultPermChecker.Cache.GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin() {
		return true, "", "admin"
	}

	env := defaultPermChecker.Cache.FindEnvironment(cluster, namespace)
	if env == nil {
		return false, "", ""
	}
	objname = env.GetName()
	action := c.Request.Method
	hasPerm, currentrole = defaultPermChecker.canDo(userAuthoriy, env.GetKind(), env.GetID(), action)
	return
}

func (defaultPermChecker *DefaultPermissionManager) HasObjectPerm(c *gin.Context, kind string, pk uint) (hasPerm bool, objname string, currentrole string) {
	user, exist := defaultPermChecker.Userif.GetContextUser(c)
	if !exist {
		hasPerm = false
		currentrole = ""
		return
	}
	userAuthoriy := defaultPermChecker.Cache.GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin() {
		hasPerm = true
		currentrole = "sysadmin"
		return
	}

	res := defaultPermChecker.Cache.FindResource(kind, pk)
	if res == nil {
		return false, "", ""
	}
	objname = res.GetName()
	action := c.Request.Method
	hasPerm, currentrole = defaultPermChecker.canDo(userAuthoriy, kind, pk, action)
	return
}

func (defaultPermChecker *DefaultPermissionManager) canDo(userAuthority *cache.UserAuthority, kind string, pk uint, action string) (hasPerm bool, currenrole string) {
	parents := defaultPermChecker.Cache.FindParents(kind, pk)
	if len(parents) == 0 {
		return true, ""
	}
	for _, res := range parents {
		switch res.GetKind() {
		case models.ResTenant:
			role := userAuthority.GetResourceRole(res.GetKind(), res.GetID())
			// 不是租户成员->直接禁止;
			// 租户管理员->放行;
			// 租户普通成员->到具体项目判断
			if role == "" {
				return false, role
			}
			if role == models.TenantRoleAdmin {
				return true, role
			}
			if slice.ContainStr(normalActions, action) {
				return true, role
			}
		case models.ResProject:
			// 不是项目成员->直接禁止;
			// 项目管理员｜项目运维->放行;
			// 项目普通成员->到具体环境判断
			role := userAuthority.GetResourceRole(res.GetKind(), res.GetID())
			if role == "" {
				return false, role
			}
			if role == models.ProjectRoleAdmin || role == models.ProjectRoleOps {
				return true, role
			}
			if slice.ContainStr(normalActions, action) {
				return true, role
			}
		case models.ResEnvironment:
			// 不是环境成员->直接禁止;
			// 环境operator->放行;
			// 环境reader->到具体动作判断
			role := userAuthority.GetResourceRole(res.GetKind(), res.GetID())
			if role == "" {
				return false, role
			}
			if role == models.EnvironmentRoleOperator {
				return true, role
			}
			if slice.ContainStr(normalActions, action) {
				return true, role
			}
		case models.ResVirtualSpace:
			// 不是虚拟空间成员->直接禁止;
			// 虚拟空间管理员->放行;
			// 虚拟空间reader->到具体动作判断
			role := userAuthority.GetResourceRole(res.GetKind(), res.GetID())
			if role == "" {
				return false, role
			}
			if role == models.VirtualSpaceRoleAdmin {
				return true, role
			}
			if slice.ContainStr(normalActions, action) {
				return true, role
			}
		}
	}
	return false, ""
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckByClusterNamespace(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	if len(namespace) == 0 {
		// 这种是处理反向代理的
		pobj, exist := c.Get("proxyobj")
		if !exist {
			return
		}
		proxyobj := pobj.(*audit.ProxyObject)
		namespace = proxyobj.Namespace
	}
	hasPerm, objname, _ := defaultPermissionChecker.HasEnvPerm(c, cluster, namespace)
	if !hasPerm {
		handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to operate the environment %s", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckByEnvironmentID(c *gin.Context) {
	envid := utils.ToUint(c.Param("environment_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResEnvironment, envid)
	if !hasPerm {
		handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to operate the environment %s", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckByProjectID(c *gin.Context) {
	projid := utils.ToUint(c.Param("project_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResProject, projid)
	if !hasPerm {
		handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to operate the project %s", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckByTenantID(c *gin.Context) {
	if c.Param("tenant_id") == "_all" {
		defaultPermissionChecker.CheckIsATenantAdmin(c)
		return
	}
	tenantid := utils.ToUint(c.Param("tenant_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResTenant, tenantid)
	if !hasPerm {
		handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to operate the tenant %s", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckByVirtualSpaceID(c *gin.Context) {
	if c.Param("virtualspace_id") == "_all" {
		defaultPermissionChecker.CheckIsSysADMIN(c)
		return
	}
	virtualspaceID := utils.ToUint(c.Param("virtualspace_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResVirtualSpace, virtualspaceID)
	if !hasPerm {
		handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to operate the virtual space %s", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckIsSysADMIN(c *gin.Context) {
	user, exist := defaultPermissionChecker.Userif.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, i18n.Errorf(c, "please login first"))
		c.Abort()
		return
	}
	userAuthoriy := defaultPermissionChecker.Cache.GetUserAuthority(user)
	if !userAuthoriy.IsSystemAdmin() {
		handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to do this operation"))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckIsATenantAdmin(c *gin.Context) {
	user, exist := defaultPermissionChecker.Userif.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, i18n.Errorf(c, "please login first"))
		c.Abort()
		return
	}
	userAuthoriy := defaultPermissionChecker.Cache.GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin() {
		return
	}
	for _, ten := range userAuthoriy.Tenants {
		if ten.IsAdmin {
			return
		}
	}
	if userAuthoriy.IsSystemAdmin() {
		return
	}
	handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to do this operation"))
	c.Abort()
}

func (defaultPermissionChecker *DefaultPermissionManager) CheckIsVirtualSpaceAdmin(c *gin.Context) {
	user, exist := defaultPermissionChecker.Userif.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, i18n.Errorf(c, "please login first"))
		c.Abort()
		return
	}
	userAuthoriy := defaultPermissionChecker.Cache.GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin() {
		return
	}
	for _, vs := range userAuthoriy.VirtualSpaces {
		if vs.IsAdmin {
			return
		}
	}
	handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to do this operation, you must be an admin in any virtual space firstly"))
	c.Abort()
}

// CheckCanDeployEnvironment 判断是否拥有环境的部署权限
// 1. 如果是系统管理员，pass
// 3. 如果是租户管理员，pass
// 3. 如果是项目管理员，pass
// 4. 如果是项目运维，pass
// 5. 如果是环境operator，pass
// 6. 其他都reject
func (defaultPermChecker *DefaultPermissionManager) CheckCanDeployEnvironment(c *gin.Context) {
	user, exist := defaultPermChecker.Userif.GetContextUser(c)
	if !exist {
		c.Abort()
		handlers.Forbidden(c, i18n.Errorf(c, "please login first"))
		return
	}
	userAuthoriy := defaultPermChecker.Cache.GetUserAuthority(user)
	// 系统管理员. pass
	if userAuthoriy.IsSystemAdmin() {
		return
	}

	envid := utils.ToUint(c.Param("environment_id"))
	if envid == 0 {
		envid = utils.ToUint(c.Query("environment_id"))
	}
	if envid == 0 {
		// 如果拿不到环境，就根据项目ID判断
		defaultPermChecker.CheckByProjectID(c)
		return
	}
	parents := defaultPermChecker.Cache.FindParents(models.ResEnvironment, envid)
	if len(parents) == 0 {
		c.Abort()
		handlers.NotOK(c, i18n.Error(c, "current environment data is abnormal, please contact the administrator"))
		return
	}

	for _, p := range parents {
		switch p.GetKind() {
		case models.ResTenant:
			// 租户管理员. pass
			if userAuthoriy.IsTenantAdmin(p.GetTenantID()) {
				return
			}
		case models.ResProject:
			// 项目管理员，项目运维. pass
			if userAuthoriy.IsProjectAdmin(p.GetProjectID()) || userAuthoriy.IsProjectOps(p.GetProjectID()) {
				return
			}
		case models.ResEnvironment:
			// 环境operator. pass
			if userAuthoriy.IsEnvironmentOperator(p.GetEnvironmentID()) {
				return
			}
		}
	}
	handlers.Forbidden(c, i18n.Errorf(c, "you have no permission to deploy in the current environment"))
	c.Abort()
}
