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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func Test_contains(t *testing.T) {
	a := assert.New(t)
	a.Equal(true, contains([]string{"1", "2"}, "1"))
	a.Equal(false, contains([]string{"1", "2"}, "3"))
}

func Test_tableName_columnName(t *testing.T) {
	a := assert.New(t)
	a.Equal(tableName("AbcDef"), "abc_defs")
	a.Equal(tableName("Abc2Def"), "abc2_defs")
	a.Equal(tableName("1Ab2c"), "1_ab2cs")
	a.Equal(tableName("1AbC2c"), "1_ab_c2cs")

	a.Equal(columnName("Model", "Column1"), "column1")
	a.Equal(columnName("Model", "column"), "column")
	a.Equal(columnName("any", "date"), "date")
}

func TestGetQuery(t *testing.T) {
	a := assert.New(t)
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				Path:     "/",
				RawQuery: "",
			},
		},
	}
	r, e := GetQuery(c, nil)
	a.Nil(e)
	a.Equal(&URLQuery{
		Page:     "1",
		Size:     "10",
		page:     1,
		size:     10,
		endPos:   10,
		preloads: []string{},
	}, r)

	c2 := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				Path:     "/",
				RawQuery: "page=1&size=2",
			},
		},
	}
	r2, e2 := GetQuery(c2, nil)
	a.Nil(e2)
	a.Equal(&URLQuery{
		Page:     "1",
		Size:     "2",
		page:     1,
		size:     2,
		endPos:   2,
		preloads: []string{},
	}, r2)

	c3 := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				Path:     "/",
				RawQuery: "page=x&size=2",
			},
		},
	}
	_, e3 := GetQuery(c3, nil)
	a.NotNil(e3)

	c4 := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				Path:     "/",
				RawQuery: "page=-1&size=-1&preload=a,b,c",
			},
		},
	}
	r4, e4 := GetQuery(c4, nil)
	a.Nil(e4)
	a.Equal(&URLQuery{
		Page:     "-1",
		Size:     "-1",
		Preload:  "a,b,c",
		page:     1,
		size:     10,
		endPos:   10,
		preloads: []string{"a", "b", "c"},
	}, r4)
}
