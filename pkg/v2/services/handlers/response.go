package handlers

import (
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/utils/pagination"
)

type Response struct {
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	ErrorData interface{} `json:"err,omitempty"`
}

type PageData struct {
	Total       int64       `json:"total"`
	List        interface{} `json:"list"`
	CurrentPage int         `json:"page"`
	CurrentSize int         `json:"size"`
}

var NewPageFromContext = pagination.NewPageDataFromContext

func Page(db *gorm.DB, total int64, data interface{}) *PageData {
	var page, size int
	p, exist := db.Get("page")
	if exist {
		page = p.(int)
	}
	s, exist := db.Get("size")
	if exist {
		size = s.(int)
	}
	return &PageData{
		Total:       total,
		List:        data,
		CurrentPage: page,
		CurrentSize: size,
	}
}

func BadRequest(resp *restful.Response, err error) {
	resp.WriteHeaderAndJson(http.StatusBadRequest, Response{Message: MessageError, ErrorData: ParseError(err)}, restful.MIME_JSON)
}

func ServiceUnavailable(resp *restful.Response, err error) {
	resp.WriteHeaderAndJson(http.StatusServiceUnavailable, err, restful.MIME_JSON)
}

func OK(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusOK, Response{Data: data, Message: MessageOK, ErrorData: nil}, restful.MIME_JSON)
}

func Created(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusCreated, Response{Data: data, Message: MessageOK, ErrorData: nil}, restful.MIME_JSON)
}

func NoContent(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusNoContent, Response{Data: data, Message: MessageOK, ErrorData: nil}, restful.MIME_JSON)
}

func Forbidden(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusForbidden, Response{Data: data, Message: MessageForbidden, ErrorData: nil}, restful.MIME_JSON)
}

func Unauthorized(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusUnauthorized, Response{Data: data, Message: MessageUnauthorized, ErrorData: nil}, restful.MIME_JSON)
}

func NotFound(resp *restful.Response, data interface{}) {
	resp.WriteHeaderAndJson(http.StatusNotFound, Response{Data: data, Message: MessageNotFound, ErrorData: nil}, restful.MIME_JSON)
}

func NotFoundOrBadRequest(resp *restful.Response, err error) {
	if IsNotFound(err) {
		NotFound(resp, err)
	} else {
		BadRequest(resp, err)
	}
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

type RespBase struct {
	Message   string      `json:"message"`
	ErrorData interface{} `json:"err"`
}

type ListBase struct {
	Total       int64 `json:"total"`
	CurrentPage int64 `json:"page"`
	CurrentSize int64 `json:"size"`
}
