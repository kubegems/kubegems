package handlers

import (
	"fmt"
	"reflect"

	restful "github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/validate"
)

func BindData(req *restful.Request, data interface{}) error {
	if err := req.ReadEntity(data); err != nil {
		return err
	}
	return validate.GetValidator().Struct(data)
}

func BindQuery(req *restful.Request, data interface{}) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("parse query error, target must be a pointer")
	}
	rv = rv.Elem()
	rt := reflect.TypeOf(data).Elem()
	for idx := 0; idx < rv.NumField(); idx++ {
		field := rv.Field(idx)
		kind := rt.Field(idx)
		var q string
		q = kind.Tag.Get("form")
		if q == "" {
			q = kind.Tag.Get("json")
		}
		field.Set(reflect.ValueOf(req.QueryParameter(q)))
	}
	return nil
}
