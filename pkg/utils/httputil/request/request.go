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

package request

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"
)

type ListOptions struct {
	Page   int    `json:"page,omitempty"`
	Size   int    `json:"size,omitempty"`
	Search string `json:"search,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// nolint: gomnd
func GetListOptions(r *http.Request) ListOptions {
	return ListOptions{
		Page:   Query(r, "page", 1),
		Size:   Query(r, "size", 10),
		Search: Query(r, "search", ""),
		Sort:   Query(r, "sort", ""),
	}
}

// nolint: forcetypeassert,gomnd,ifshort
func Query[T any](r *http.Request, key string, defaultValue T) T {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	switch any(defaultValue).(type) {
	case string:
		return any(val).(T)
	case []string:
		if val == "" {
			return defaultValue
		}
		return any(strings.Split(val, ",")).(T)
	case int:
		intval, _ := strconv.Atoi(val)
		return any(intval).(T)
	case bool:
		b, _ := strconv.ParseBool(val)
		return any(b).(T)
	case int64:
		intval, _ := strconv.ParseInt(val, 10, 64)
		return any(intval).(T)
	default:
		return defaultValue
	}
}

func Body(r *http.Request, into any) error {
	switch r.Header.Get("Content-Type") {
	case "application/json", "":
		return json.NewDecoder(r.Body).Decode(into)
	case "application/xml":
		return xml.NewDecoder(r.Body).Decode(into)
	}
	return nil
}
