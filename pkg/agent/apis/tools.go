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

package apis

import (
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"kubegems.io/kubegems/pkg/utils/pagination"
)

const (
	allNamespace = "_all"
)

func getLabelSelector(c *gin.Context) labels.Selector {
	labelsMap := c.QueryMap("labels")
	sel := labels.SelectorFromSet(labelsMap)
	return sel
}

func getDefaultHeader(c *gin.Context, key, defaultV string) string {
	value := c.Request.Header.Get(key)
	if len(value) == 0 {
		return defaultV
	}
	return value
}

var NewPageDataFromContext = pagination.NewPageDataFromContext

type SortAndSearchAble = pagination.SortAndSearchAble

func getFieldSelector(c *gin.Context) (fields.Selector, bool) {
	fieldSelectorStr := c.Query("fieldSelector")
	if len(fieldSelectorStr) == 0 {
		return nil, false
	}
	sel, err := fields.ParseSelector(fieldSelectorStr)
	if err != nil {
		return nil, false
	}
	return sel, true
}

func getFieldSelectorMatch(c *gin.Context) (map[string]string, bool) {
	ret := map[string]string{}
	sel, exist := getFieldSelector(c)
	if !exist {
		return ret, false
	}
	for _, req := range sel.Requirements() {
		if req.Operator == selection.Equals {
			ret[req.Field] = req.Value
		}
	}
	return ret, len(ret) > 0
}
