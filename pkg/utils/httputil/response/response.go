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
	"fmt"
	"io"
	"net/http"
)

type Response struct {
	Message string `json:"message,omitempty"` // user friendly message, it contains error message or success message
	Data    any    `json:"data,omitempty"`    // data
	Error   any    `json:"error,omitempty"`   // raw error for debug purpose only
}

func OK(w http.ResponseWriter, data any) {
	Raw(w, http.StatusOK, Response{Data: data}, nil)
}

func NotFound(w http.ResponseWriter, message string) {
	Error(w, NewStatusErrorMessage(http.StatusNotFound, message))
}

func BadRequest(w http.ResponseWriter, message string) {
	Error(w, NewStatusErrorMessage(http.StatusBadRequest, message))
}

func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, NewStatusErrorMessage(http.StatusUnauthorized, message))
}

func Forbidden(w http.ResponseWriter, message string) {
	Error(w, NewStatusErrorMessage(http.StatusForbidden, message))
}

func InternalServerError(w http.ResponseWriter, err error) {
	Error(w, NewStatusError(http.StatusInternalServerError, err))
}

var ServerError = InternalServerError

func Error(w http.ResponseWriter, err error) {
	statusError := &StatusError{}
	if errors.As(err, &statusError) {
		Raw(w, statusError.Status, Response{Message: statusError.Error(), Error: statusError.RawErr}, nil)
	} else {
		Raw(w, http.StatusBadRequest, Response{Message: err.Error(), Error: err}, nil)
	}
}

func Raw(w http.ResponseWriter, status int, data any, headers map[string]string) {
	for k, v := range headers {
		w.Header().Set(k, v)
	}
	switch val := data.(type) {
	case io.Reader:
		setContentTypeIfNotSet(w.Header(), "application/octet-stream")
		w.WriteHeader(status)
		_, _ = io.Copy(w, val)
	case string:
		setContentTypeIfNotSet(w.Header(), "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(val))
	case []byte:
		setContentTypeIfNotSet(w.Header(), "application/octet-stream")
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

func setContentTypeIfNotSet(hds http.Header, val string) {
	if val := hds.Get("Content-Type"); val == "" {
		hds.Set("Content-Type", val)
	}
}

type StatusError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	RawErr  error  `json:"error,omitempty"`
}

func (e StatusError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.RawErr != nil {
		return e.RawErr.Error()
	}
	return http.StatusText(e.Status)
}

func NewStatusErrorMessage(status int, message string) *StatusError {
	return &StatusError{Status: status, Message: message}
}

// NewStatusErrorf acts like fmt.Errorf but returns a StatusError.
// Usage:
//
//	if err:=someprocess(username); err!=nil {
//	  return NewStatusErrorf(http.StatusNotFound, "user %s not found: %w", username, err)
//	}
func NewStatusErrorf(status int, format string, args ...any) *StatusError {
	err := fmt.Errorf(format, args...)
	return &StatusError{Status: status, Message: err.Error(), RawErr: errors.Unwrap(err)}
}

func NewStatusError(status int, err error) *StatusError {
	return &StatusError{Status: status, Message: err.Error(), RawErr: err}
}
