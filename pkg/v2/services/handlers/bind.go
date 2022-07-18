// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"fmt"
	"reflect"

	restful "github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/v2/model/validate"
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
