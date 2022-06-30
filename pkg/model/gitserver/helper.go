package gitserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func OK(w http.ResponseWriter, data interface{}) {
	RawResponse(w, http.StatusOK, nil, data)
}

func Created(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

func BadRequest(w http.ResponseWriter, data interface{}) {
	RawResponse(w, http.StatusBadRequest, nil, data)
}

func NotFound(w http.ResponseWriter) {
	RawResponse(w, http.StatusNotFound, nil, nil)
}

func InternalServerError(w http.ResponseWriter, data interface{}) {
	RawResponse(w, http.StatusInternalServerError, nil, data)
}

func RawResponse(w http.ResponseWriter, code int, header map[string]string, body interface{}) {
	for k, v := range header {
		w.Header().Set(k, v)
	}
	switch val := body.(type) {
	case string:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(code)
		w.Write([]byte(val))
	case io.Reader:
		w.WriteHeader(code)
	case []byte:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(code)
		w.Write(val)
	case nil:
		w.WriteHeader(code)
	default:
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}
}

func SetHeaderNoCache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func SetHeaderCacheForever(w http.ResponseWriter) {
	now := time.Now().Unix()
	w.Header().Set("Date", fmt.Sprintf("%d", now))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}

func HeadersMap(headers http.Header) map[string]string {
	m := make(map[string]string)
	for k, v := range headers {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}
