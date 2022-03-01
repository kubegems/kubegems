package userhandler

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/handlers/base"
	"kubegems.io/pkg/utils"
)

var userTags = []string{"users"}

type Handler struct {
	base.BaseHandler
}

func (h *Handler) CreateUser(req *restful.Request, resp *restful.Response) {
	user := &models.UserCreate{}
	if err := handlers.BindData(req, user); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	pass, _ := utils.MakePassword(user.Password)
	newUser := &models.User{
		Email:        user.Email,
		Username:     user.Username,
		Password:     pass,
		SystemRoleID: 2,
	}
	if err := h.DB().WithContext(req.Request.Context()).Create(newUser).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	user.ID = newUser.ID
	handlers.Created(resp, user)
}

func (h *Handler) ListUser(req *restful.Request, resp *restful.Response) {
	userList := []models.UserCommon{}
	if err := h.DB().WithContext(req.Request.Context()).Find(userList).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, userList)
}

func (h *Handler) RetrieveUser(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	user := models.UserCommon{}
	if err := h.DB().WithContext(ctx).First(user, "username = ?", req.PathParameter("name")).Error; err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
	}
	handlers.OK(resp, user)
}

func (h *Handler) DeleteUser(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	user := models.UserCommon{}
	if err := h.DB().WithContext(ctx).Delete(user, "username = ?", req.PathParameter("name")).Error; err != nil {
		if handlers.IsNotFound(err) {
			handlers.NoContent(resp, nil)
		} else {
			handlers.BadRequest(resp, err)
		}
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) PutUser(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	user := &models.UserCommon{}
	if err := h.DB().WithContext(ctx).First(user, "username = ?", req.PathParameter("name")).Error; err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
	}
	newUser := &models.UserCommon{}
	if err := handlers.BindData(req, newUser); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	user.Email = newUser.Email
	user.Phone = newUser.Phone
	if err := h.DB().WithContext(ctx).Save(user).Error; err != nil {
		handlers.BadRequest(resp, err)
	}
	handlers.OK(resp, user)

}
