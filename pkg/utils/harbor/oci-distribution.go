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

/*
* OCI Distribution Specification Client
*
* For more information visit below URL
* https://github.com/opencontainers/distribution-spec/blob/main/spec.md#endpoints
*
 */
package harbor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	specsv1 "github.com/opencontainers/distribution-spec/specs-go/v1"
)

type OCIDistributionClient struct {
	Server   string
	Username string
	Password string
}

func NewOCIDistributionClient(server, username, password string) *OCIDistributionClient {
	return &OCIDistributionClient{Server: server, Username: username, Password: password}
}

// end-8a	GET	/v2/<name>/tags/list
func (c *OCIDistributionClient) ListTags(ctx context.Context, image string) (*specsv1.TagList, error) {
	_, path, name, _, _ := ParseImag(image)
	fullpath := path + "/" + name
	tags := &specsv1.TagList{}
	err := c.request(ctx, http.MethodGet, "/v2/"+fullpath+"/tags/list", nil, tags)
	return tags, err
}

// end-8a

// 参考OCI规范此段实现 https://github.com/opencontainers/distribution-spec/blob/main/spec.md#determining-support
// 目前大部分(所有)镜像仓库均实现了OCI Distribution 规范，可以使用 /v2 接口进行推断，
// 如果认证成功则返回200则认为实现了OCI且认证成功
// end-1	GET	/v2/	200	404/401
func (c *OCIDistributionClient) Ping(ctx context.Context) error {
	err := c.request(ctx, http.MethodGet, "/v2", nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *OCIDistributionClient) request(ctx context.Context, method string, path string, postbody interface{}, into interface{}) error {
	var body io.Reader
	switch typed := postbody.(type) {
	// convert to bytes
	case []byte:
		body = bytes.NewBuffer(typed)
	// thise type can processed by 'http.NewRequestWithContext(...)'
	case io.Reader:
		body = typed
	case nil:
		// do nothing
	// send json format
	default:
		bts, err := json.Marshal(postbody)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(bts)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.Server+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		errresp := &specsv1.ErrorResponse{}
		if err := json.NewDecoder(resp.Body).Decode(errresp); err != nil {
			return err
		}
		return errorResponseError(errresp)
	}
	if into != nil {
		return json.NewDecoder(resp.Body).Decode(into)
	}
	return nil
}

func errorResponseError(err *specsv1.ErrorResponse) error {
	if err == nil {
		return nil
	}

	msg := err.Error() + ":"
	for _, e := range err.Detail() {
		msg += e.Message + ";"
	}
	return errors.New(msg)
}
