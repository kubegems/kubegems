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

package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sort"
)

type Response struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type Page struct {
	List  interface{} `json:"list,omitempty"`
	Total int64       `json:"total,omitempty"`
	Page  int64       `json:"page,omitempty"`
	Size  int64       `json:"size,omitempty"`
}

type PageFilterFunc func(i int) bool

type PageSortFunc func(i, j int) bool

const defaultPageSize = 10

func NewPageData(list interface{}, page, size int, filterfn PageFilterFunc, sortfn PageSortFunc) Page {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}
	// sort
	if sortfn != nil {
		sort.Slice(list, sortfn)
	}

	v := reflect.ValueOf(list)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return Page{}
	}

	// filter
	if filterfn != nil {
		ret := reflect.MakeSlice(v.Type(), 0, size)
		for i := 0; i < v.Len(); i++ {
			if filterfn(i) {
				ret = reflect.Append(ret, v.Index(i))
			}
		}
		v = ret
	}

	// page
	total := v.Len()
	start := (page - 1) * size
	end := page * size
	if end > total {
		end = total
	}
	v = v.Slice(start, end)

	return Page{
		List:  v.Interface(),
		Total: int64(total),
		Page:  int64(page),
		Size:  int64(size),
	}
}

func OK(w http.ResponseWriter, data interface{}) {
	DoRawResponse(w, http.StatusOK, data, nil)
}

func BadRequest(w http.ResponseWriter, message string) {
	ErrorResponse(w, StatusError{Status: http.StatusBadRequest, Message: message})
}

func ServerError(w http.ResponseWriter, err error) {
	ErrorResponse(w, StatusError{Status: http.StatusInternalServerError, Message: err.Error()})
}

func ErrorResponse(w http.ResponseWriter, err error) {
	serr := &StatusError{}
	if errors.As(err, &serr) {
		DoRawResponse(w, serr.Status, Response{Message: err.Error(), Error: err}, nil)
	} else {
		DoRawResponse(w, http.StatusBadRequest, Response{Message: err.Error(), Error: err}, nil)
	}
}

func DoRawResponse(w http.ResponseWriter, status int, data interface{}, headers map[string]string) {
	for k, v := range headers {
		w.Header().Set(k, v)
	}
	switch val := data.(type) {
	case io.Reader:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(status)
		_, _ = io.Copy(w, val)
	case string:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(val))
	case []byte:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(status)
		_, _ = w.Write(val)
	case nil:
		w.WriteHeader(status)
		// do not write a nil representation
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(data)
	}
}

type StatusError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (e StatusError) Error() string {
	return e.Message
}

func NewError(status int, message string) *StatusError {
	return &StatusError{
		Status:  status,
		Message: message,
	}
}
