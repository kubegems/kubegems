package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
)

type SingleResponseStruct struct {
	Message   string
	Data      interface{}
	ErrorData interface{}
}

type HttpClient struct {
	BaseAddr string
	*http.Client
}

func NewHttpClientFrom(c *WrappedClient) *HttpClient {
	return &HttpClient{
		BaseAddr: c.BaseAddr.String(),
		Client: &http.Client{
			Transport: c.transport.Clone(),
			Timeout:   c.Timeout,
		},
	}
}

type ListResponseStruct struct {
	Message   string
	Data      PageData
	ErrorData interface{}
}

type PageData struct {
	Total       int64
	List        interface{}
	CurrentPage int64
	CurrentSize int64
}

type Path struct {
	gvk       schema.GroupVersionKind
	namespace string
	name      string
}

func gpath() *Path {
	return &Path{}
}

func (p *Path) WithGVK(gvk *schema.GroupVersionKind) *Path {
	p.gvk = *gvk
	return p
}

func (p *Path) WithNS(ns *string) *Path {
	if ns != nil {
		p.namespace = *ns
	}
	return p
}

func (p *Path) WithName(name *string) *Path {
	if name != nil {
		p.name = *name
	}
	return p
}

func (p Path) String() string {
	if p.gvk.Group == "" {
		p.gvk.Group = "core"
	}
	if len(p.namespace) == 0 {
		if len(p.name) == 0 {
			return fmt.Sprintf("/v1/%s/%s/%s", p.gvk.Group, p.gvk.Version, p.gvk.Kind)
		} else {
			return fmt.Sprintf("/v1/%s/%s/%s/%s", p.gvk.Group, p.gvk.Version, p.gvk.Kind, p.name)
		}
	} else {
		if len(p.name) == 0 {
			return fmt.Sprintf("/v1/%s/%s/namespaces/%s/%s", p.gvk.Group, p.gvk.Version, p.namespace, p.gvk.Kind)
		} else {
			return fmt.Sprintf("/v1/%s/%s/namespaces/%s/%s/%s", p.gvk.Group, p.gvk.Version, p.namespace, p.gvk.Kind, p.name)
		}
	}
}

func (c *HttpClient) GetObject(gvk *schema.GroupVersionKind, data interface{}, namespace, name *string) error {
	path := gpath().WithGVK(gvk).WithNS(namespace).WithName(name).String()
	uri := c.BaseAddr + path
	resp, err := c.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	response := &SingleResponseStruct{Data: data}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return toKubernetesErr(resp, gvk, name, response.Message)
	}
	return nil
}

func (c *HttpClient) GetObjectList(gvk *schema.GroupVersionKind, data interface{}, namespace *string, labelSel map[string]string) error {
	path := gpath().WithGVK(gvk).WithNS(namespace).String()
	q := url.Values{}
	q.Add("page", "1")
	q.Add("size", "10000")
	map2query(&q, labelSel)
	uri := c.BaseAddr + path + "?" + q.Encode()
	resp, err := c.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	response := ListResponseStruct{Data: PageData{List: data}}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return toKubernetesErr(resp, gvk, nil, response.Message)
	}
	return nil
}

func (c *HttpClient) CreateObject(gvk *schema.GroupVersionKind, data interface{}, namespace, name *string) error {
	path := gpath().WithGVK(gvk).WithNS(namespace).WithName(name).String()
	uri := c.BaseAddr + path
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(dataBytes)
	resp, err := c.Post(uri, jsonContentType, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	response := &SingleResponseStruct{Data: data}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return toKubernetesErr(resp, gvk, name, response.Message)
	}
	return nil
}

func (c *HttpClient) PatchObject(gvk *schema.GroupVersionKind, data interface{}, namespace, name *string) error {
	path := gpath().WithGVK(gvk).WithNS(namespace).WithName(name).String()
	uri := c.BaseAddr + path
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(dataBytes)
	req, err := http.NewRequest(http.MethodPatch, uri, body)
	req.Header.Set("Content-Type", jsonContentType)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	response := &SingleResponseStruct{Data: data}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return toKubernetesErr(resp, gvk, name, response.Message)
	}
	return nil
}

func (c *HttpClient) UpdateObject(gvk *schema.GroupVersionKind, data interface{}, namespace, name *string) error {
	path := gpath().WithGVK(gvk).WithNS(namespace).WithName(name).String()
	uri := c.BaseAddr + path
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(dataBytes)
	req, err := http.NewRequest(http.MethodPut, uri, body)
	req.Header.Set("Content-Type", jsonContentType)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	response := &SingleResponseStruct{Data: data}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return toKubernetesErr(resp, gvk, name, response.Message)
	}
	return nil
}

func (c *HttpClient) DeleteObject(gvk *schema.GroupVersionKind, namespace, name *string) error {
	path := gpath().WithGVK(gvk).WithNS(namespace).WithName(name).String()
	uri := c.BaseAddr + path
	req, err := http.NewRequest(http.MethodDelete, uri, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	response := &SingleResponseStruct{}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return toKubernetesErr(resp, gvk, name, response.Message)
	}
	return nil
}

func map2query(q *url.Values, m map[string]string) {
	for k, v := range m {
		q.Set("labels["+k+"]", v)
	}
}

func toKubernetesErr(resp *http.Response, gvk *schema.GroupVersionKind, name *string, errmsg string) *errors.StatusError {
	return errors.NewGenericServerResponse(
		resp.StatusCode,
		resp.Request.Method,
		schema.GroupResource{
			Group:    gvk.Group,
			Resource: gvk.Kind,
		},
		pointer.StringDeref(name, ""),
		errmsg,
		0,
		true)
}
