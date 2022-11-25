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

package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const jsonContentType = "application/json"

func NewSimpleTypedClient(baseaddr string) (*TypedClient, error) {
	agenturl, err := url.Parse(baseaddr)
	if err != nil {
		return nil, err
	}
	return &TypedClient{
		BaseAddr:      agenturl,
		RuntimeScheme: kube.GetScheme(),
		HTTPClient:    http.DefaultClient,
	}, nil
}

type TypedClient struct {
	BaseAddr      *url.URL
	RuntimeScheme *runtime.Scheme
	HTTPClient    *http.Client
}

var _ client.WithWatch = TypedClient{}

func (c TypedClient) RESTMapper() meta.RESTMapper {
	panic("not implemented") // TODO: Implement
}

func (c TypedClient) Scheme() *runtime.Scheme {
	return c.RuntimeScheme
}

func (c TypedClient) Status() client.StatusWriter {
	return &StatusTypedClient{c: c}
}

type StatusTypedClient struct {
	c TypedClient
}

func (c *StatusTypedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	queries := map[string]string{"subresource": "true"}
	return c.c.request(ctx, http.MethodPut, jsonContentType, obj, obj.GetNamespace(), obj.GetName(), queries, nil)
}

func (c *StatusTypedClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	options := client.PatchOptions{}
	options.ApplyOptions(opts)

	queries := map[string]string{"subresource": "true"}
	if options.Force != nil {
		queries["force"] = strconv.FormatBool(*options.Force)
	}

	patchcontent, err := patch.Data(obj)
	if err != nil {
		return err
	}
	return c.c.request(ctx, http.MethodPatch, string(patch.Type()), obj, obj.GetNamespace(), obj.GetName(), queries, patchcontent)
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (c TypedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return c.request(ctx, http.MethodGet, jsonContentType, obj, key.Namespace, key.Name, nil, nil)
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (c TypedClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	options := client.ListOptions{}
	options.ApplyOptions(opts)

	queries := c.listOptionToQueries(opts)
	return c.request(ctx, http.MethodGet, jsonContentType, list, options.Namespace, "", queries, nil)
}

func (c TypedClient) listOptionToQueries(opts []client.ListOption) map[string]string {
	options := client.ListOptions{}
	options.ApplyOptions(opts)

	queries := map[string]string{
		"continue": options.Continue,
		"limit":    strconv.Itoa(int(options.Limit)),
	}

	if options.LabelSelector != nil {
		queries["label-selector"] = options.LabelSelector.String()
	}

	if options.FieldSelector != nil {
		queries["field-selector"] = options.FieldSelector.String()
	}
	return queries
}

// Create saves the object obj in the Kubernetes cluster.
func (c TypedClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return c.request(ctx, http.MethodPost, jsonContentType, obj, obj.GetNamespace(), obj.GetName(), nil, nil)
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c TypedClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	options := client.PatchOptions{}
	options.ApplyOptions(opts)

	queries := make(map[string]string)

	if options.Force != nil {
		queries["force"] = strconv.FormatBool(*options.Force)
	}
	queries["field-manager"] = options.FieldManager
	patchcontent, err := patch.Data(obj)
	if err != nil {
		return err
	}
	return c.request(ctx, http.MethodPatch, string(patch.Type()), obj, obj.GetNamespace(), obj.GetName(), queries, patchcontent)
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c TypedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return c.request(ctx, http.MethodPut, jsonContentType, obj, obj.GetNamespace(), obj.GetName(), nil, nil)
}

// Delete deletes the given obj from Kubernetes cluster.
func (c TypedClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	options := client.DeleteOptions{}
	options.ApplyOptions(opts)

	queries := map[string]string{}
	if options.GracePeriodSeconds != nil {
		queries["grace-period-seconds"] = strconv.Itoa(int(*options.GracePeriodSeconds))
	}
	if options.PropagationPolicy != nil {
		queries["propagation-policy"] = string(*options.PropagationPolicy)
	}
	return c.request(ctx, http.MethodDelete, jsonContentType, obj, obj.GetNamespace(), obj.GetName(), nil, nil)
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (c TypedClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	panic("not implemented") // TODO: Implement
}

func (c TypedClient) request(ctx context.Context, method, contenttype string,
	obj runtime.Object, namespace, name string, queries map[string]string, data []byte,
) error {
	addr, err := c.requestAddr(obj, method, namespace, name, queries)
	if err != nil {
		return err
	}

	var body io.Reader
	if method != http.MethodGet {
		if data != nil {
			body = bytes.NewReader(data)
		} else {
			content, err := json.Marshal(obj)
			if err != nil {
				return err
			}
			body = bytes.NewReader(content)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, addr, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contenttype)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// err
	if resp.StatusCode >= http.StatusBadRequest {
		status := &metav1.Status{}
		if err := json.NewDecoder(resp.Body).Decode(status); err != nil {
			return err
		}
		return &errors.StatusError{ErrStatus: *status}
	}

	// ok
	if err := json.NewDecoder(resp.Body).Decode(obj); err != nil {
		return err
	}
	return nil
}

func (c TypedClient) requestAddr(obj runtime.Object, method string, namespace, name string, queries map[string]string) (string, error) {
	gvk, err := apiutil.GVKForObject(obj, c.RuntimeScheme)
	if err != nil {
		return "", err
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	obj.GetObjectKind().SetGroupVersionKind(gvk)

	sb := &strings.Builder{}
	// assumes without a suffix '/'
	sb.WriteString(c.BaseAddr.String())
	sb.WriteString("/internal")
	if gvk.Group == "" {
		sb.WriteString("/core")
	} else {
		sb.WriteString("/" + gvk.Group)
	}
	sb.WriteString("/" + gvk.Version)
	if namespace != "" {
		sb.WriteString("/namespaces")
		sb.WriteString("/" + namespace)
	}
	sb.WriteString("/" + gvk.Kind)

	if method != http.MethodPost && name != "" {
		sb.WriteString("/" + name)
	}

	vals := &url.Values{}
	for k, v := range queries {
		vals.Set(k, v)
	}

	return sb.String() + "?" + vals.Encode(), nil
}

/*
不知道如何正确使用 watch.Interface，参考k8s源码:
	https://github.com/kubernetes/kubernetes/blob/release-1.20/pkg/volume/csi/csi_attacher.go#L444-L487

或者：

	watcher, err := cli.Watch(ctx, objctList)
	if err != nil {
		return fmt.Errorf("watch error:%v", err)
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				log.Warningf("watch channel had been closed")
				break
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				...
			case watch.Deleted:
				...
			case watch.Error:
				log.Warningf("received watch error: %v", event)
			}

		case <-ctx.Done():
			log.Warningf("watch channel closed")
			break
		}
	}
*/
//
func (c TypedClient) Watch(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	options := client.ListOptions{}
	options.ApplyOptions(opts)

	queries := c.listOptionToQueries(opts)

	// list as watch
	queries["watch"] = "true"

	gvk, err := apiutil.GVKForObject(obj, c.RuntimeScheme)
	if err != nil {
		return nil, err
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	addr, err := c.requestAddr(obj, http.MethodGet, options.Namespace, "", queries)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// err
	if resp.StatusCode >= http.StatusBadRequest {
		status := &metav1.Status{}
		if err := json.NewDecoder(resp.Body).Decode(status); err != nil {
			return nil, err
		}
		return nil, &errors.StatusError{ErrStatus: *status}
	}

	newitemfunc := func() client.Object {
		obj, err := c.RuntimeScheme.New(gvk)
		if err != nil {
			return &unstructured.Unstructured{}
		}
		return obj.(client.Object)
	}

	return watch.NewStreamWatcher(
		NewRestDecoder(resp.Body, newitemfunc),
		// use 500 to indicate that the cause of the error is unknown - other error codes
		// are more specific to HTTP interactions, and set a reason
		errors.NewClientErrorReporter(http.StatusInternalServerError, "", "ClientWatchDecoding")), nil
}

type restDecoder struct {
	r           io.ReadCloser
	jd          *json.Decoder
	newitemfunc func() client.Object
}

func NewRestDecoder(r io.ReadCloser, newitemfunc func() client.Object) *restDecoder {
	return &restDecoder{
		r:           r,
		jd:          json.NewDecoder(r),
		newitemfunc: newitemfunc,
	}
}

func (d *restDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	obj := &watch.Event{Object: d.newitemfunc()}
	if err := d.jd.Decode(obj); err != nil {
		return watch.Error, obj.Object, err
	}
	return obj.Type, obj.Object, nil
}

func (d *restDecoder) Close() {
	d.r.Close()
}
