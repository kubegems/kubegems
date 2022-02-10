package userhandler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/service/handlers"
	"github.com/kubegems/gems/pkg/utils"

	"github.com/gin-gonic/gin"
)

var (
	SearchFields   = []string{"Username", "Email"}
	FilterFields   = []string{"Username"}
	PreloadFields  = []string{"Tenants", "SystemRole"}
	OrderFields    = []string{"Username", "ID"}
	ModelName      = "User"
	PrimaryKeyName = "user_id"
)

// ListUser 列表 User
// @Tags User
// @Summary User列表
// @Description User列表
// @Accept json
// @Produce json
// @Param Username query string false "Username"
// @Param preload query string false "choices Tenants,SystemRole"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (Username,Email)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "User"
// @Router /v1/user [get]
// @Security JWT
func (h *UserHandler) ListUser(c *gin.Context) {
	var list []models.User
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:         "User",
		PreloadFields: PreloadFields,
		SearchFields:  SearchFields,
		SortFields:    OrderFields,
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// RetrieveUser User详情
// @Tags User
// @Summary User详情
// @Description get User详情
// @Accept json
// @Produce json
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "User"
// @Router /v1/user/{user_id} [get]
// @Security JWT
func (h *UserHandler) RetrieveUser(c *gin.Context) {
	var obj models.User
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, obj)
}

// PostUser 创建User
// @Tags User
// @Summary 创建User
// @Description 创建User
// @Accept json
// @Produce json
// @Param param body models.User true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "User"
// @Router /v1/user [post]
// @Security JWT
func (h *UserHandler) PostUser(c *gin.Context) {
	var obj models.User
	if err := c.BindJSON(&obj); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := h.GetDB().Create(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "创建", "系统用户", obj.Username)
	handlers.Created(c, obj)
}

// PutUser 修改User
// @Tags User
// @Summary 修改User
// @Description 修改User，目前只能修改Email、Phone
// @Accept json
// @Produce json
// @Param user_id path uint true "user_id"
// @Param param body models.User true "表单"
// @Success 200 {object} handlers.ResponseStruct{Data=models.User} "User"
// @Router /v1/user/{user_id} [put]
// @Security JWT
func (h *UserHandler) PutUser(c *gin.Context) {
	var oldUser, newUser models.User
	if err := h.GetDB().First(&oldUser, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := c.BindJSON(&newUser); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if strconv.Itoa(int(newUser.ID)) != c.Param(PrimaryKeyName) {
		handlers.NotOK(c, fmt.Errorf("请求体参数ID和URL参数ID不一致"))
		return
	}

	oldUser.Email = newUser.Email
	oldUser.Phone = newUser.Phone

	if err := h.GetDB().Save(&oldUser).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}

	h.SetAuditData(c, "更新", "系统用户", oldUser.Username)

	handlers.OK(c, oldUser)
}

// DeleteUser 删除 User
// @Tags User
// @Summary 删除 User
// @Description 删除 User
// @Accept json
// @Produce json
// @Param user_id path uint true "user_id"
// @Success 204 {object} handlers.ResponseStruct "resp"
// @Router /v1/user/{user_id} [delete]
// @Security JWT
func (h *UserHandler) DeleteUser(c *gin.Context) {
	var obj models.User
	if err := h.GetDB().First(&obj, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NoContent(c, nil)
		return
	}
	if err := h.GetDB().Delete(&obj).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	h.SetAuditData(c, "删除", "系统用户", obj.Username)
	handlers.NoContent(c, nil)
}

// ListUserTenant 获取属于User的 Tenant 列表
// @Tags User
// @Summary 获取属于 User 的 Tenant 列表
// @Description 获取属于 User 的 Tenant 列表
// @Accept json
// @Produce json
// @Param user_id path uint true "user_id"
// @Param preload query string false "choices ResourceQuotas,Users,Projects"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (TenantName,Remark)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.Tenant}} "models.Tenant"
// @Router /v1/user/{user_id}/tenant [get]
// @Security JWT
func (h *UserHandler) ListUserTenant(c *gin.Context) {
	var list []models.Tenant

	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	query.Order = "TenantName"
	cond := &handlers.PageQueryCond{
		Model:         "Tenant",
		PreloadFields: []string{"ResourceQuotas", "Users", "Projects"},
		SearchFields:  []string{"TenantName", "Remark"},
		Join:          handlers.Args("join tenant_user_rels on tenant_user_rels.tenant_id = tenants.id"),
		Where:         []*handlers.QArgs{handlers.Args("tenant_user_rels.user_id = ?", c.Param(PrimaryKeyName))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

// ResetUserPassword 重置用户密码
// @Tags User
// @Summary 重置用户密码
// @Description 重置用户密码
// @Accept json
// @Produce json
// @Param user_id path uint true "user_id"
// @Success 200 {object} handlers.ResponseStruct{Data=resetPasswordResult} "data"
// @Router /v1/user/{user_id}/reset_password [post]
// @Security JWT
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	var user models.User
	if err := h.GetDB().First(&user, c.Param(PrimaryKeyName)).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	passwd := utils.GeneratePassword()
	password, err := utils.MakePassword(passwd)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	user.Password = password
	if err := h.GetDB().Save(&user).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, &resetPasswordResult{Password: passwd})
}

// ListEnvironmentUser 获取多个环境的用户列表
// @Tags User
// @Summary 获取多个环境的用户列表
// @Description 获取多个环境的用户列表
// @Accept json
// @Produce json
// @Param environment_id path uint true "环境id，中间以逗号隔开"
// @Param page query int false "page"
// @Param size query int false "page"
// @Param search query string false "search in (Username,Email)"
// @Success 200 {object} handlers.ResponseStruct{Data=handlers.PageData{List=[]models.User}} "models.User"
// @Router /v1/user/_/environment/{environment_id} [get]
// @Security JWT
func (h *UserHandler) ListEnvironmentUser(c *gin.Context) {
	var list []models.User
	query, err := handlers.GetQuery(c, nil)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cond := &handlers.PageQueryCond{
		Model:        "User",
		SearchFields: []string{"Username", "Email"},
		Select:       handlers.Args("users.*, environment_user_rels.role"),
		Join:         handlers.Args("join environment_user_rels on environment_user_rels.user_id = users.id"),
		Where:        []*handlers.QArgs{handlers.Args("environment_user_rels.environment_id in ?", strings.Split(c.Param("environment_id"), ","))},
	}
	total, page, size, err := query.PageList(h.GetDB(), cond, &list)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, handlers.Page(total, list, int64(page), int64(size)))
}

type resetPasswordResult struct {
	Password string `json:"password"`
}
