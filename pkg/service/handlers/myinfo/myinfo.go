package myinfohandler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
)

// Myinfo 获取当前用户的信息
// @Tags         User
// @Summary      获取当前用户的信息
// @Description  获取当前用户的信息
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.ResponseStruct{Data=[]models.User}  "用户详情"
// @Router       /v1/my/info [get]
// @Security     JWT
func (h *MyHandler) Myinfo(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, fmt.Errorf("请登录"))
		return
	}
	var user models.User
	if e := h.GetDB().Preload("SystemRole").Preload("Tenants").First(&user, "id = ?", u.GetID()).Error; e != nil {
		handlers.Forbidden(c, fmt.Errorf("请重新登录"))
		return
	}
	handlers.OK(c, user)
}

// MyAuthority 获取当前用户权限列表
// @Tags         User
// @Summary      获取当前用户权限列表
// @Description  获取当前用户权限列表
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.ResponseStruct{}  "用户权限列表"
// @Router       /v1/my/auth [get]
// @Security     JWT
func (h *MyHandler) MyAuthority(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, fmt.Errorf("请登录"))
		return
	}

	auth := h.ModelCache().FlushUserAuthority(u)
	handlers.OK(c, auth)
}

// MyTenants 获取当前用户租户列表
// @Tags         User
// @Summary      获取当前用户租户列表
// @Description  获取当前用户租户列表
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.ResponseStruct{}  "用户权限列表"
// @Router       /v1/my/tenants [get]
// @Security     JWT
func (h *MyHandler) MyTenants(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, fmt.Errorf("请登录"))
		return
	}
	tenants := []models.Tenant{}
	h.GetDB().
		Joins("tenant_user_rels on tenant_user_rels.tenant_id = tenants.id").
		Where("tenant_user_rels.user_id = ?", u.GetID()).Find(&tenants)
	handlers.OK(c, tenants)
}

// ResetPassword 重设密码
// @Tags         User
// @Summary      重设密码
// @Description  重设密码
// @Accept       json
// @Produce      json
// @Param        param  body      resetPasswordForm          true  "表单"
// @Success      200    {object}  handlers.ResponseStruct{}  ""
// @Router       /v1/my/reset_password [post]
// @Security     JWT
func (h *MyHandler) ResetPassword(c *gin.Context) {
	u, exist := h.GetContextUser(c)
	if !exist {
		handlers.Unauthorized(c, fmt.Errorf("请登录"))
		return
	}
	cuser := models.User{}
	h.GetDB().First(&cuser, u.GetID())
	form := &resetPasswordForm{}
	c.BindJSON(form)

	if form.New1 != form.New2 {
		handlers.NotOK(c, fmt.Errorf("两次输入密码不一致"))
		return
	}

	if err := utils.ValidPassword(form.New1); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := utils.ValidatePassword(form.Origin, cuser.Password); err != nil {
		fmt.Println(form.Origin)
		fmt.Println(cuser.Password)
		handlers.NotOK(c, fmt.Errorf("原始密码错误"))
		return
	}

	pass, err := utils.MakePassword(form.New1)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	cuser.Password = pass
	if err := h.GetDB().Save(&cuser).Error; err != nil {
		return
	}
	handlers.OK(c, nil)
}

// GetMyConfig 获取用户配置
// @Tags         System
// @Summary      获取用户配置
// @Description  获取用户配置
// @Accept       json
// @Produce      json
// @Param        name  path      string                                             true  "配置名"
// @Success      200   {object}  handlers.ResponseStruct{Data=models.OnlineConfig}  "resp"
// @Router       /v1/my/config/{name} [get]
// @Security     JWT
func (h *MyHandler) GetMyConfig(c *gin.Context) {
	cfg := models.OnlineConfig{}
	if err := h.GetDB().First(&cfg, "name = ?", c.Param("name")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, cfg)
}

type resetPasswordForm struct {
	Origin string `json:"origin"`
	New1   string `json:"new1"`
	New2   string `json:"new2"`
}
