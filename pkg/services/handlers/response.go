package handlers

import (
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/model/client"
)

type ResponseStruct struct {
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	ErrorData interface{} `json:"err"`
}

type PageData struct {
	Total       int64       `json:"total"`
	List        interface{} `json:"list"`
	CurrentPage int64       `json:"page"`
	CurrentSize int64       `json:"size"`
}

func Page(l client.ObjectListIface, data interface{}) *PageData {
	page, size := l.GetPageSize()
	return &PageData{
		Total:       *l.GetTotal(),
		List:        data,
		CurrentPage: *page,
		CurrentSize: *size,
	}
}

func BadRequest(resp *restful.Response, err error) {
	resp.WriteHeaderAndJson(http.StatusBadRequest, ParseError(err), restful.MIME_JSON)
}

func OK(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusOK, data, restful.MIME_JSON)
}

func Created(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusCreated, data, restful.MIME_JSON)
}

func NoContent(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusNoContent, data, restful.MIME_JSON)
}

func Forbidden(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusForbidden, data, restful.MIME_JSON)
}

func Unauthorized(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusUnauthorized, data, restful.MIME_JSON)
}

func NotFound(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusNotFound, data, restful.MIME_JSON)
}

func NotFoundOrBadRequest(resp *restful.Response, err error) {
	if err == gorm.ErrRecordNotFound {
		NotFound(resp, err)
	} else {
		BadRequest(resp, err)
	}
}
