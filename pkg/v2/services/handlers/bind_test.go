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
	"net/http"
	"net/url"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
)

func TestBindQuery(t *testing.T) {
	req := restful.NewRequest(&http.Request{
		Method: "GET",
		URL: &url.URL{
			Host:     "baidu.com",
			Path:     "/xx/x",
			RawQuery: "a=1&b=2",
		},
	})
	type Q struct {
		A string `form:"a"`
		B string `form:"b"`
	}

	q := Q{}
	if err := BindQuery(req, q); err != nil {
		t.Error(err)
	} else {
		t.Log(q)
	}
}
