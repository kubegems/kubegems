package cache

import (
	"encoding/json"

	"kubegems.io/kubegems/pkg/service/models"
)

type UserResource struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	IsAdmin bool   `json:"isAdmin"`
}

// UserAuthority 用户权限
type UserAuthority struct {
	SystemRole    string          `json:"systemRole"`
	Tenants       []*UserResource `json:"tenant"`
	Projects      []*UserResource `json:"projects"`
	Environments  []*UserResource `json:"environments"`
	VirtualSpaces []*UserResource `json:"virtualSpaces"`
}

func (auth *UserAuthority) MarshalBinary() ([]byte, error) {
	return json.Marshal(auth)
}

func (auth *UserAuthority) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &auth)
}

func (auth *UserAuthority) GetResourceRole(kind string, id uint) string {
	switch kind {
	case models.ResTenant:
		for _, tenant := range auth.Tenants {
			if id == uint(tenant.ID) {
				return tenant.Role
			}
		}
	case models.ResProject:
		for _, proj := range auth.Projects {
			if id == uint(proj.ID) {
				return proj.Role
			}
		}
	case models.ResEnvironment:
		for _, env := range auth.Environments {
			if id == uint(env.ID) {
				return env.Role
			}
		}
	case models.ResVirtualSpace:
		for _, vs := range auth.VirtualSpaces {
			if id == uint(vs.ID) {
				return vs.Role
			}
		}
	}
	return ""
}

func (auth *UserAuthority) IsAnyTenantAdmin() bool {
	for _, t := range auth.Tenants {
		if t.Role == models.TenantRoleAdmin {
			return true
		}
	}
	return false
}

func (auth *UserAuthority) IsSystemAdmin() bool {
	return auth.SystemRole == models.SystemRoleAdmin
}

func (auth *UserAuthority) IsTenantAdmin(tenantid uint) bool {
	role := auth.GetResourceRole(models.ResTenant, tenantid)
	return role == models.TenantRoleAdmin
}

func (auth *UserAuthority) IsTenantMember(tenantid uint) bool {
	role := auth.GetResourceRole(models.ResTenant, tenantid)
	return role == models.TenantRoleOrdinary
}

func (auth *UserAuthority) IsProjectAdmin(projectid uint) bool {
	role := auth.GetResourceRole(models.ResProject, projectid)
	return role == models.ProjectRoleAdmin
}

func (auth *UserAuthority) IsProjectDev(projectid uint) bool {
	role := auth.GetResourceRole(models.ResProject, projectid)
	return role == models.ProjectRoleDev
}

func (auth *UserAuthority) IsProjectTest(projectid uint) bool {
	role := auth.GetResourceRole(models.ResProject, projectid)
	return role == models.ProjectRoleTest
}

func (auth *UserAuthority) IsProjectOps(projectid uint) bool {
	role := auth.GetResourceRole(models.ResProject, projectid)
	return role == models.ProjectRoleOps
}

func (auth *UserAuthority) IsEnvironmentOperator(envid uint) bool {
	role := auth.GetResourceRole(models.ResEnvironment, envid)
	return role == models.EnvironmentRoleOperator
}

func (auth *UserAuthority) IsEnvironmentReader(envid uint) bool {
	role := auth.GetResourceRole(models.ResEnvironment, envid)
	return role == models.EnvironmentRoleReader
}

func (auth *UserAuthority) IsVirtualSpaceAdmin(vsid uint) bool {
	role := auth.GetResourceRole(models.ResVirtualSpace, vsid)
	return role == models.VirtualSpaceRoleAdmin
}

func (auth *UserAuthority) IsVirtualSpaceMember(vsid uint) bool {
	role := auth.GetResourceRole(models.ResVirtualSpace, vsid)
	return role == models.VirtualSpaceRoleNormal
}
