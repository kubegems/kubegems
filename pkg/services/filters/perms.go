package filters

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/auth/user"
	"kubegems.io/pkg/utils/slice"
)

type PermMiddleware struct {
	DB *gorm.DB
}

func NewPermMiddleware(db *gorm.DB) *PermMiddleware {
	return &PermMiddleware{DB: db}
}

func (p *PermMiddleware) FilterFunc(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if IsWhiteList(req) {
		chain.ProcessFilter(req, resp)
		return
	}
	u := req.Attribute("user")
	if u == nil {
		resp.WriteHeaderAndJson(http.StatusForbidden, "no user", restful.MIME_JSON)
	}
	user, ok := u.(user.CommonUserIface)
	if !ok {
		resp.WriteHeaderAndJson(http.StatusForbidden, "error user", restful.MIME_JSON)
		return
	}
	if !p.hasPerm(user, req) {
		resp.WriteHeaderAndJson(http.StatusForbidden, "no perms", restful.MIME_JSON)
		return
	}
	chain.ProcessFilter(req, resp)
}

func (p *PermMiddleware) hasPerm(user user.CommonUserIface, req *restful.Request) bool {
	if p.hasPermForMetaData(user, req) {
		return true
	}
	// TODO : cluster agent api perm check
	return true
}

func (p *PermMiddleware) hasPermForMetaData(user user.CommonUserIface, req *restful.Request) bool {
	ctx := req.Request.Context()
	params := req.PathParameters()
	tenantName, existTenant := params["tenant"]
	if !existTenant {
		return false
	}
	// 1. get tenant role
	// 2. not belong to tenant -> reject
	// 3. tenant admin -> pass
	// 4. tenant member -> continue
	tenantUserRel := &models.TenantUserRels{}
	if err := p.DB.WithContext(ctx).
		Joins("LEFT JOIN tenants on tenants.id = tenant_user_rels.tenant_id").
		Where("tenants.name = ?", tenantName).
		Where("tenant_user_rels.user_id = ?", user.GetID()).
		First(tenantUserRel).Error; err != nil {
		return false
	}
	if tenantUserRel.Role == models.TenantRoleAdmin {
		return true
	}
	projectName, existProject := params["project"]
	if !existProject {
		if req.Request.Method == http.MethodGet {
			return true
		} else {
			return false
		}
	}

	// 1. get project role
	// 2. not belong to project -> reject
	// 3. project admin -> pass
	// 4. project member -> continue
	project := &models.Project{}
	if err := p.DB.WithContext(ctx).
		Joins("LEFT JOIN tenants on tenants.id = projects.tenant_id").
		Where("tenants.name = ?", tenantName).
		Where("projects.name = ?", projectName).
		First(project).Error; err != nil {
		return false
	}
	projectUserRel := &models.ProjectUserRels{}
	if err := p.DB.WithContext(ctx).
		Where("project_id = ?", project.ID).
		Where("user_id = ?", user.GetID()).
		First(projectUserRel); err != nil {
		return false
	}
	projectRole := projectUserRel.Role
	if projectRole == models.ProjectRoleAdmin || projectRole == models.ProjectRoleOps {
		return true
	}
	environmentName, existEnvironment := params["environment"]
	if !existEnvironment {
		if req.Request.Method == http.MethodGet {
			return true
		} else {
			return false
		}
	}

	// 1. env metatype is dev -> allow  all project members any op
	// 2. env metatype is test -> allow project [test, ops, admin] any op
	// 3. env metatype is prod -> allow project [ops, admin] any op
	// 4. if user is env op -> allow any
	environment := &models.Environment{}
	if err := p.DB.WithContext(ctx).
		Where("project_id = ?", project.ID).
		Where("name = ?", environmentName).
		First(environment).Error; err != nil {
		return false
	}

	// projectMembers can read all environment
	if req.Request.Method == http.MethodGet {
		return true
	}
	// dev environment allows any role operate
	if environment.MetaType == models.EnvironmentMetaTypeDev {
		return true
	}
	// test environment allows roles ["test", "admin", "ops"] operate
	if environment.MetaType == models.EnvironmentMetaTypeTest && slice.ContainStr([]string{models.ProjectRoleTest, models.ProjectRoleOps, models.ProjectRoleAdmin}, projectRole) {
		return true
	}
	// prod environment allows roles ["admin", "ops"] operate
	if environment.MetaType == models.EnvironmentMetaTypeProd && slice.ContainStr([]string{models.ProjectRoleOps, models.ProjectRoleAdmin}, projectRole) {
		return true
	}
	envUserRel := &models.EnvironmentUserRels{}
	if err := p.DB.WithContext(ctx).
		Where("user_id = ?", user.GetID()).
		Where("environment_id = ?", environment.ID).
		First(envUserRel).Error; err != nil {
		return false
	}
	if envUserRel.Role == models.EnvironmentRoleOperator {
		return true
	}
	return false
}
