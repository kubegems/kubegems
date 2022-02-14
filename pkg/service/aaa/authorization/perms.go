package authorization

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/aaa/audit"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
)

var normalActions = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
}

// PermissionChecker 权限判断工具,仅支持租户，项目，环境三级数据
type PermissionChecker interface {
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

type CacheLayer interface {
	GetCacheLayer() *models.CacheLayer
	GetContextUser(c *gin.Context) (*models.User, bool)
}

type DefaultPermissionChecker struct {
	CacheLayer
}

func (defaultPermChecker *DefaultPermissionChecker) HasEnvPerm(c *gin.Context, cluster, namespace string) (hasPerm bool, objname string, currentrole string) {
	user, exist := defaultPermChecker.GetContextUser(c)
	if !exist {
		return false, "", ""
	}
	userAuthoriy := defaultPermChecker.GetCacheLayer().GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin {
		return true, "", "admin"
	}

	resourceTree := defaultPermChecker.GetCacheLayer().GetGlobalResourceTree()
	env := resourceTree.Tree.FindNodeByClusterNamespace(cluster, namespace)
	if env == nil {
		return false, "", ""
	}
	objname = env.Name
	action := c.Request.Method
	hasPerm, currentrole = defaultPermChecker.canDo(userAuthoriy, env.Kind, env.ID, action)
	return
}

func (defaultPermChecker *DefaultPermissionChecker) HasObjectPerm(c *gin.Context, kind string, pk uint) (hasPerm bool, objname string, currentrole string) {
	user, exist := defaultPermChecker.GetContextUser(c)
	if !exist {
		hasPerm = false
		currentrole = ""
		return
	}
	userAuthoriy := defaultPermChecker.GetCacheLayer().GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin {
		hasPerm = true
		currentrole = "sysadmin"
		return
	}

	resourceTree := defaultPermChecker.GetCacheLayer().GetGlobalResourceTree()
	res := resourceTree.Tree.FindNode(kind, pk)
	if res == nil {
		return false, "", ""
	}
	objname = res.Name
	action := c.Request.Method
	hasPerm, currentrole = defaultPermChecker.canDo(userAuthoriy, kind, pk, action)
	return
}

func (defaultPermChecker *DefaultPermissionChecker) canDo(userAuthority *models.UserAuthority, kind string, pk uint, action string) (hasPerm bool, currenrole string) {
	resourceTree := defaultPermChecker.GetCacheLayer().GetGlobalResourceTree()
	parents := resourceTree.Tree.FindParents(kind, pk)
	if len(parents) == 0 {
		return true, ""
	}
	for _, res := range parents {
		switch res.Kind {
		case models.ResTenant:
			role := userAuthority.GetResourceRole(res.Kind, res.ID)
			// 不是租户成员->直接禁止;
			// 租户管理员->放行;
			// 租户普通成员->到具体项目判断
			if role == "" {
				return false, role
			}
			if role == models.TenantRoleAdmin {
				return true, role
			}
			if utils.ContainStr(normalActions, action) {
				return true, role
			}
		case models.ResProject:
			// 不是项目成员->直接禁止;
			// 项目管理员->放行;
			// 项目普通成员->到具体环境判断
			role := userAuthority.GetResourceRole(res.Kind, res.ID)
			if role == "" {
				return false, role
			}
			if role == models.ProjectRoleAdmin {
				return true, role
			}
			if utils.ContainStr(normalActions, action) {
				return true, role
			}
		case models.ResEnvironment:
			// 不是环境成员->直接禁止;
			// 环境operator->放行;
			// 环境reader->到具体动作判断
			role := userAuthority.GetResourceRole(res.Kind, res.ID)
			if role == "" {
				return false, role
			}
			if role == models.EnvironmentRoleOperator {
				return true, role
			}
			if utils.ContainStr(normalActions, action) {
				return true, role
			}
		case models.ResVirtualSpace:
			// 不是虚拟空间成员->直接禁止;
			// 虚拟空间管理员->放行;
			// 虚拟空间reader->到具体动作判断
			role := userAuthority.GetResourceRole(res.Kind, res.ID)
			if role == "" {
				return false, role
			}
			if role == models.VirtualSpaceRoleAdmin {
				return true, role
			}
			if utils.ContainStr(normalActions, action) {
				return true, role
			}
		}
	}
	return false, ""
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckByClusterNamespace(c *gin.Context) {
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
		handlers.Forbidden(c, fmt.Sprintf("当前用户没有环境%s的操作权限", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckByEnvironmentID(c *gin.Context) {
	envid := utils.ToUint(c.Param("environment_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResEnvironment, envid)
	if !hasPerm {
		handlers.Forbidden(c, fmt.Sprintf("当前用户没有环境%s的操作权限", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckByProjectID(c *gin.Context) {
	projid := utils.ToUint(c.Param("project_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResProject, projid)
	if !hasPerm {
		handlers.Forbidden(c, fmt.Sprintf("当前用户没有项目%s的操作权限", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckByTenantID(c *gin.Context) {
	if c.Param("tenant_id") == "_all" {
		defaultPermissionChecker.CheckIsATenantAdmin(c)
		return
	}
	tenantid := utils.ToUint(c.Param("tenant_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResTenant, tenantid)
	if !hasPerm {
		handlers.Forbidden(c, fmt.Sprintf("当前用户没有租户%s的操作权限", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckByVirtualSpaceID(c *gin.Context) {
	if c.Param("virtualspace_id") == "_all" {
		defaultPermissionChecker.CheckIsSysADMIN(c)
		return
	}
	virtualspaceID := utils.ToUint(c.Param("virtualspace_id"))
	hasPerm, objname, _ := defaultPermissionChecker.HasObjectPerm(c, models.ResVirtualSpace, virtualspaceID)
	if !hasPerm {
		handlers.Forbidden(c, fmt.Sprintf("当前用户没有虚拟空间%s的操作权限", objname))
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckIsSysADMIN(c *gin.Context) {
	user, exist := defaultPermissionChecker.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, "请登录")
		c.Abort()
		return
	}
	userAuthoriy := defaultPermissionChecker.GetCacheLayer().GetUserAuthority(user)
	if !userAuthoriy.IsSystemAdmin {
		handlers.Forbidden(c, "只有系统管理员可以执行当前操作")
		c.Abort()
		return
	}
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckIsATenantAdmin(c *gin.Context) {
	user, exist := defaultPermissionChecker.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, "请登录")
		c.Abort()
		return
	}
	userAuthoriy := defaultPermissionChecker.GetCacheLayer().GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin {
		return
	}
	for _, ten := range userAuthoriy.Tenants {
		if ten.IsAdmin {
			return
		}
	}
	if userAuthoriy.IsSystemAdmin {
		return
	}
	handlers.Forbidden(c, "租户管理员才能执行当前操作")
	c.Abort()
}

func (defaultPermissionChecker *DefaultPermissionChecker) CheckIsVirtualSpaceAdmin(c *gin.Context) {
	user, exist := defaultPermissionChecker.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, "请登录")
		c.Abort()
		return
	}
	userAuthoriy := defaultPermissionChecker.GetCacheLayer().GetUserAuthority(user)
	if userAuthoriy.IsSystemAdmin {
		return
	}
	for _, vs := range userAuthoriy.VirtualSpaces {
		if vs.IsAdmin {
			return
		}
	}
	handlers.Forbidden(c, "虚拟空间管理员才能执行当前操作")
	c.Abort()
}

// CheckCanDeployEnvironment 判断是否拥有环境的部署权限
// 1. 如果是系统管理员，pass
// 3. 如果是租户管理员，pass
// 3. 如果是项目管理员，pass
// 4. 如果是项目运维，pass
// 5. 如果是环境operator，pass
// 6. 其他都reject
func (defaultPermChecker *DefaultPermissionChecker) CheckCanDeployEnvironment(c *gin.Context) {
	user, exist := defaultPermChecker.GetContextUser(c)
	if !exist {
		c.Abort()
		handlers.Forbidden(c, "请登录")
		return
	}
	userAuthoriy := defaultPermChecker.GetCacheLayer().GetUserAuthority(user)
	// 系统管理员. pass
	if userAuthoriy.IsSystemAdmin {
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
	resourceTree := defaultPermChecker.GetCacheLayer().GetGlobalResourceTree()
	parents := resourceTree.Tree.FindParents(models.ResEnvironment, envid)
	if len(parents) == 0 {
		c.Abort()
		handlers.NotOK(c, fmt.Errorf("当前环境数据异常，请联系管理员"))
		return
	}

	for _, p := range parents {
		switch p.Kind {
		case models.ResTenant:
			// 租户管理员. pass
			if userAuthoriy.IsTenantAdmin(p.ID) {
				return
			}
		case models.ResProject:
			// 项目管理员，项目运维. pass
			if userAuthoriy.IsProjectAdmin(p.ID) || userAuthoriy.IsProjectOps(p.ID) {
				return
			}
		case models.ResEnvironment:
			// 环境operator. pass
			if userAuthoriy.IsEnvironmentOperator(p.ID) {
				return
			}
		}
	}
	c.Abort()
	handlers.Forbidden(c, "你没有当前环境的部署权限")
}
