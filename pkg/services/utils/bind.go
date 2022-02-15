package utils

import (
	restful "github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/validate"
)

func BindData(req *restful.Request, data interface{}) error {
	if err := req.ReadEntity(data); err != nil {
		return err
	}
	return validate.GetValidator().Struct(data)
}
