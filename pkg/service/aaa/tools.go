package aaa

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/models"
)

type UserInterface interface {
	SetContextUser(c *gin.Context, user *models.User)
	GetContextUser(c *gin.Context) (*models.User, bool)
}

type UserInfoHandler struct {
	ContextUserKey string
}

func NewUserInfoHandler() *UserInfoHandler {
	return &UserInfoHandler{
		ContextUserKey: "current_user",
	}
}

func (i *UserInfoHandler) SetContextUser(c *gin.Context, user *models.User) {
	c.Set(i.ContextUserKey, user)
}

func (i *UserInfoHandler) GetContextUser(c *gin.Context) (*models.User, bool) {
	user, exist := c.Get(i.ContextUserKey)
	if exist {
		return user.(*models.User), true
	}
	return nil, false
}
