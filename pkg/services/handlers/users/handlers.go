package userhandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/utils"
)

var (
	userTags = []string{"users"}
)

type Handler struct {
	Path        string
	ModelClient client.ModelClientIface
}

func (h *Handler) Create(req *restful.Request, resp *restful.Response) {
	user := &forms.UserDetail{}
	if err := utils.BindData(req, user); err != nil {
		utils.BadRequest(resp, err)
		return
	}
	if err := h.ModelClient.Create(req.Request.Context(), user.AsObject()); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(user)
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	userList := forms.UserCommonList{}
	l := userList.AsListObject()
	if err := h.ModelClient.List(req.Request.Context(), l, handlers.CommonOptions(req)...); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteAsJson(handlers.PageList(l, userList.AsListData()))
}

func (h *Handler) Retrieve(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "retrieve"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Delete(req *restful.Request, resp *restful.Response) {
	uname := req.PathParameter("username")
	uid := req.PathParameter("id")
	user := &forms.UserCommon{}
	var cond *client.WhereOption
	if uname != "" {
		cond = client.Where("username", client.Eq, uname)
	} else {
		cond = client.Where("id", client.Eq, uid)
	}
	if err := h.ModelClient.Delete(req.Request.Context(), user.AsObject(), cond); err != nil {
		resp.WriteError(http.StatusBadRequest, err)
		return
	}
	resp.WriteHeaderAndEntity(http.StatusNoContent, nil)
}

func (h *Handler) Put(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "put"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Patch(req *restful.Request, resp *restful.Response) {
	msg := map[string]interface{}{"status": "patch"}
	resp.WriteAsJson(msg)
}

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/" + h.Path)
	ws.Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("/").
		To(h.List).
		Doc("list users").
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, handlers.PageData{
			List: []forms.UserCommon{},
		})))

	ws.Route(ws.POST("/").
		To(h.Create).
		Doc("create user").
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Reads(forms.UserDetail{}).
		Returns(http.StatusOK, handlers.MessageOK, forms.UserCommon{}))

	ws.Route(ws.DELETE("/{username:^\\w+$}").
		To(h.Delete).
		Doc("delete user via usernmae").
		Param(restful.PathParameter("username", "username")).
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.DELETE("/{id:^\\d+$}").
		To(h.Delete).
		Doc("delete user via id").
		Param(restful.PathParameter("id", "id")).
		Metadata(restfulspec.KeyOpenAPITags, userTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	container.Add(ws)
}

type User struct {
	Username string
}
