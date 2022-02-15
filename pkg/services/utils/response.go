package utils

import (
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
)

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
