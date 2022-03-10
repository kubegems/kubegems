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
