package systemrolehandler

import (
	"fmt"

	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/msgbus"

	"github.com/gin-gonic/gin"
	msgclient "kubegems.io/pkg/msgbus/client"
)

var (
	ModelName      = "SystemRole"
	SearchFields   = []string{"RoleName"}
	OrderFields    = []string{}
	PrimaryKeyName = "systemrole_id"
)

// ListSystemRole 列表 SystemRole
// @Tags SystemRole
// @Summary SystemRole列表
// @Description SystemRole列表
// @Accept json
// @Produce json
// @Param RoleName query string false "RoleName"
// @Param preload query string false "choices Users"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (RoleName)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.SystemRole}} "SystemRole"
// @Router /v1/systemrole [get]
// @Security JWT
func (h *SystemRoleHandler) ListSystemRole(c *gin.Context) {
	var list []models.SystemRole
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         ModelName,
		SearchFields:  SearchFields,
		SortFields:    OrderFields,
		PreloadFields: []string{"Users"},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveSystemRole SystemRole详情
// @Tags SystemRole
// @Summary SystemRole详情
// @Description get SystemRole详情
// @Accept json
// @Produce json
// @Param systemrole_id path uint true "systemrole_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.SystemRole} "SystemRole"
// @Router /v1/systemrole/{systemrole_id} [get]
// @Security JWT
func (h *SystemRoleHandler) RetrieveSystemRole(c *gin.Context) {
	var obj models.SystemRole
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PostSystemRole 创建SystemRole
// @Tags SystemRole
// @Summary 创建SystemRole
// @Description 创建SystemRole
// @Accept json
// @Produce json
// @Param param body models.SystemRole true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.SystemRole} "SystemRole"
// @Router /v1/systemrole [post]
// @Security JWT
func (h *SystemRoleHandler) PostSystemRole(c *gin.Context) {
	var obj models.SystemRole
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Create(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "系统角色", obj.RoleName)
	handlers.Created(c, obj)
}

// DeleteSystemRole 删除 SystemRole
// @Tags SystemRole
// @Summary 删除 SystemRole
// @Description 删除 SystemRole
// @Accept json
// @Produce json
// @Param systemrole_id path uint true "systemrole_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/systemrole/{systemrole_id} [delete]
// @Security JWT
func (h *SystemRoleHandler) DeleteSystemRole(c *gin.Context) {
	var obj models.SystemRole
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, nil)
		return
	}
	if err := h.GetDB().Delete(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "删除", "系统角色", obj.RoleName)
	handlers.NoContent(c, nil)
}

// ListSystemRoleUser 获取属于SystemRole的 User 列表
// @Tags SystemRole
// @Summary 获取属于 SystemRole 的 User 列表
// @Description 获取属于 SystemRole 的 User 列表
// @Accept json
// @Produce json
// @Param systemrole_id path uint true "systemrole_id"
// @Param preload query string false "choices Tenants,SystemRole"
// @Param page query int false "page"
// @Param size query int false "page"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "models.User"
// @Router /v1/systemrole/{systemrole_id}/user [get]
// @Security JWT
func (h *SystemRoleHandler) ListSystemRoleUser(c *gin.Context) {
	var (
		obj  models.SystemRole
		list []models.User
	)
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "Users",
		PreloadFields: []string{"Tenants", "SystemRole"},
		Where:         []*handlers.QArgs{handlers.Args("system_role_id = ?", obj.ID)},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, page, size))
}

// PutSystemRoleUser 修改 User 和 SystemRole 的关联关系
// @Tags SystemRole
// @Summary  修改 User 和 SystemRole 的关联关系
// @Description  修改 User 和 SystemRole 的关联关系
// @Accept json
// @Produce json
// @Param systemrole_id path uint true "systemrole_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/systemrole/{systemrole_id}/user/{user_id} [put]
// @Security JWT
func (h *SystemRoleHandler) PutSystemRoleUser(c *gin.Context) {
	var (
		role models.SystemRole
		user models.User
	)
	roleid := c.Param("systemrole_id")
	userid := c.Param("user_id")
	if err := h.GetDB().First(&role, roleid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().First(&user, userid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	user.SystemRoleID = role.ID
	if err := h.GetDB().Model(&user).Save(&user).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetCacheLayer().FlushUserAuthority(&user)
	h.SetAuditData(c, "授权", "用户系统角色", fmt.Sprintf("角色[%v]/用户[%v]", role.RoleName, user.Username))

	h.SendToMsgbus(c, func(msg *msgclient.MsgRequest) {
		msg.EventKind = msgbus.Update
		msg.ResourceType = msgbus.User
		msg.ResourceID = user.ID
		msg.Detail = fmt.Sprintf("将用户%s的系统角色设置为了%s", user.Username, role.RoleName)
		msg.ToUsers.
			Append(user.ID). // 自己
			Append(func() []uint {
				if role.RoleCode == models.SystemRoleAdmin {
					return h.GetDataBase().SystemAdmins()
				}
				return nil
			}()...) // 系统管理员
		msg.AffectedUsers.Append(user.ID) // 环境所有用户刷新权限
	})

	handlers.OK(c, user)
}

// DeleteSystemRoleUser 删除 User 和 SystemRole 的关系
// @Tags SystemRole
// @Summary 删除 User 和 SystemRole 的关系
// @Description 删除 User 和 SystemRole 的关系
// @Accept json
// @Produce json
// @Param systemrole_id path uint true "systemrole_id"
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "models.User"
// @Router /v1/systemrole/{systemrole_id}/user/{user_id} [delete]
// @Security JWT
func (h *SystemRoleHandler) DeleteSystemRoleUser(c *gin.Context) {
	var (
		role models.SystemRole
		user models.User
	)
	roleid := c.Param("systemrole_id")
	userid := c.Param("user_id")
	if err := h.GetDB().First(&role, roleid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().First(&user, userid).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.GetCacheLayer().FlushUserAuthority(&user)
	h.SetAuditData(c, "取消授权", "用户系统角色", fmt.Sprintf("角色[%v]/用户[%v]", role.RoleName, user.Username))

	if err := h.GetDB().Model(&role).Association("Users").Delete(&user); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, user)
}
