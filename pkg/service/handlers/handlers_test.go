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
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewPageData(t *testing.T) {
	type args struct {
		list     interface{}
		page     int
		size     int
		filterfn PageFilterFunc
		sortfn   PageSortFunc
	}
	filterfnData := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	filterfn := func(i int) bool { return filterfnData[i] == 2 }

	sortfnData := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	sortfn := func(i, j int) bool { return sortfnData[i] > sortfnData[j] }

	sortfilterfnData := []string{"1", "11", "111", "222", "333", "1111", "11111"}
	sortfn1 := func(i, j int) bool { return len(sortfilterfnData[i]) > len(sortfilterfnData[j]) }
	filterfn1 := func(i int) bool { return len(sortfilterfnData[i]) == 3 }

	tests := []struct {
		name string
		args args
		want PageData
	}{
		{
			name: "Normal with invalid data",
			args: args{
				list:     "123",
				page:     1,
				size:     2,
				filterfn: nil,
				sortfn:   nil,
			},
			want: PageData{},
		}, {
			name: "Normal with slice data",
			args: args{
				list:     []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}[:],
				page:     1,
				size:     2,
				filterfn: nil,
				sortfn:   nil,
			},
			want: PageData{
				CurrentPage: 1,
				CurrentSize: 2,
				Total:       10,
				List:        []int{1, 2},
			},
		}, {
			name: "Normal with pointer data",
			args: args{
				list:     &[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
				page:     1,
				size:     2,
				sortfn:   nil,
				filterfn: nil,
			},
			want: PageData{
				CurrentPage: 1,
				CurrentSize: 2,
				Total:       10,
				List:        []int{1, 2},
			},
		}, {
			name: "Normal with invalida page size",
			args: args{
				list:     &[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
				page:     1,
				size:     -1,
				sortfn:   nil,
				filterfn: nil,
			},
			want: PageData{
				CurrentPage: 1,
				CurrentSize: 10,
				Total:       10,
				List:        []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
			},
		}, {
			name: "Normal with filterfn",
			args: args{
				list:     filterfnData,
				page:     1,
				size:     10,
				sortfn:   nil,
				filterfn: filterfn,
			},
			want: PageData{
				CurrentPage: 1,
				CurrentSize: 10,
				Total:       1,
				List:        []int{2},
			},
		}, {
			name: "Normal with sortfn",
			args: args{
				list:     sortfnData,
				page:     1,
				size:     2,
				sortfn:   sortfn,
				filterfn: nil,
			},
			want: PageData{
				CurrentPage: 1,
				CurrentSize: 2,
				Total:       10,
				List:        []int{9, 8},
			},
		}, {
			name: "Normal with sortfn & filterfn",
			args: args{
				list:     sortfilterfnData,
				page:     1,
				size:     2,
				sortfn:   sortfn1,
				filterfn: filterfn1,
			},
			want: PageData{
				CurrentPage: 1,
				CurrentSize: 2,
				Total:       3,
				List:        []string{"333", "222"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPageData(tt.args.list, tt.args.page, tt.args.size, tt.args.filterfn, tt.args.sortfn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPageData() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
