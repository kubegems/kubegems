package aaa

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/models"
)

type ContextUserGetter interface {
	GetContextUser(c *gin.Context) (models.CommonUserIface, bool)
}

type ContextUserSetter interface {
	SetContextUser(c *gin.Context, user models.CommonUserIface)
}
type ContextUserOperator interface {
	ContextUserGetter
	ContextUserSetter
}

type UserInfoHandler struct {
	ContextUserKey string
}

func NewUserInfoHandler() *UserInfoHandler {
	return &UserInfoHandler{
		ContextUserKey: "current_user",
	}
}

func (i *UserInfoHandler) SetContextUser(c *gin.Context, user models.CommonUserIface) {
	c.Set(i.ContextUserKey, user)
}

func (i *UserInfoHandler) GetContextUser(c *gin.Context) (models.CommonUserIface, bool) {
	user, exist := c.Get(i.ContextUserKey)
	if exist {
		return user.(models.CommonUserIface), true
	}
	return nil, false
}
