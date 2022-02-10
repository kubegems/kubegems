package agents

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/utils/proxy"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const jsonContentType = "application/json"

func NewTypedClientFrom(client *Client) *TypedClient {
	return &TypedClient{
		serveraddr: client.BaseAddr.String(),
		http:       &http.Client{Transport: client.transport.Clone()},
		scheme:     scheme.Scheme,
	}
}

type TypedClient struct {
	scheme     *runtime.Scheme
	http       *http.Client
	serveraddr string
}

var _ client.Client = &TypedClient{}

func (c *TypedClient) RESTMapper() meta.RESTMapper {
	panic("not implemented") // TODO: Implement
}

func (c *TypedClient) Scheme() *runtime.Scheme {
	return c.scheme
}

func (c *TypedClient) Status() client.StatusWriter {
	return &StatusTypedClient{c: c}
}

type StatusTypedClient struct {
	c *TypedClient
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
func (c *TypedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return c.request(ctx, http.MethodGet, jsonContentType, obj, key.Namespace, key.Name, nil, nil)
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (c *TypedClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	options := client.ListOptions{}
	options.ApplyOptions(opts)

	queries := c.listOptionToQueries(opts)
	return c.request(ctx, http.MethodGet, jsonContentType, list, options.Namespace, "", queries, nil)
}

func (c *TypedClient) listOptionToQueries(opts []client.ListOption) map[string]string {
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
func (c *TypedClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return c.request(ctx, http.MethodPost, jsonContentType, obj, obj.GetNamespace(), obj.GetName(), nil, nil)
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *TypedClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	options := client.PatchOptions{}
	options.ApplyOptions(opts)

	queries := make(map[string]string)

	if options.Force != nil {
		queries["force"] = strconv.FormatBool(*options.Force)
	}

	patchcontent, err := patch.Data(obj)
	if err != nil {
		return err
	}
	return c.request(ctx, http.MethodPatch, string(patch.Type()), obj, obj.GetNamespace(), obj.GetName(), queries, patchcontent)
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *TypedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return c.request(ctx, http.MethodPut, jsonContentType, obj, obj.GetNamespace(), obj.GetName(), nil, nil)
}

// Delete deletes the given obj from Kubernetes cluster.
func (c *TypedClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
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
func (c *TypedClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	panic("not implemented") // TODO: Implement
}

func (c *TypedClient) request(ctx context.Context, method, contenttype string,
	obj runtime.Object, namespace, name string, queries map[string]string, data []byte) error {
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

	resp, err := c.http.Do(req)
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

func (c *TypedClient) requestAddr(obj runtime.Object, method string, namespace, name string, queries map[string]string) (string, error) {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return "", err
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	sb := &strings.Builder{}
	// assumes without a suffix '/'
	sb.WriteString(c.serveraddr)
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

type SingleResponseStruct struct {
	Message   string
	Data      interface{}
	ErrorData interface{}
}

func (c *TypedClient) DoRawRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	addr := c.serveraddr + path
	req, err := http.NewRequestWithContext(ctx, method, addr, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	// err
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		response := &SingleResponseStruct{}
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("request error: %s", response.Message)
	}
	return resp, nil
}

func (c *TypedClient) DoRequest(ctx context.Context, method, path string, body io.Reader, into interface{}) error {
	resp, err := c.DoRawRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// must success
	response := &SingleResponseStruct{Data: into}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	return nil
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
func (c *TypedClient) Watch(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	options := client.ListOptions{}
	options.ApplyOptions(opts)

	queries := c.listOptionToQueries(opts)

	// list as watch
	queries["watch"] = "true"

	gvk, err := apiutil.GVKForObject(obj, c.scheme)
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

	resp, err := c.http.Do(req)
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
		obj, err := c.scheme.New(gvk)
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

type PortForwarder struct {
	ctx            context.Context
	cancel         context.CancelFunc
	httpRequestUrl *url.URL
	httpreq        *http.Request
	ln             net.Listener
}

func newPortForwarder(ctx context.Context, target string) (*PortForwarder, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	p := &PortForwarder{
		ctx:            ctx,
		cancel:         cancel,
		httpRequestUrl: u,
		httpreq:        req,
	}

	if err := p.start(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PortForwarder) start() error {
	// listen
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	p.ln = ln

	log.Infof("port forwarder listening on %s", ln.Addr().String())

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			select {
			case <-p.ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Errorf("error accepting connection: %v", err)
				continue
			}
			go p.servconn(conn)
		}
	}()

	return nil
}

func (p *PortForwarder) servconn(conn net.Conn) {
	dst, err := net.Dial("tcp", p.httpRequestUrl.Host)
	if err != nil {
		log.Errorf("error dial: %v", err)
		return
	}
	// open as http
	if err := p.httpreq.Write(dst); err != nil {
		log.Errorf("error wrilte request: %v", err)
		return
	}
	// using as tcp
	if err := proxy.CopyDuplex(conn, dst, -1); err != nil {
		log.Errorf("error copy duplex: %v", err)
	}
}

func (p *PortForwarder) ListenAddr() net.Addr {
	return p.ln.Addr()
}

func (p *PortForwarder) Stop() {
	p.cancel()
	p.ln.Close()
}

//  PortForward
// Deprecated: 无法使用，因 service 与 agent 中间还有一层 http proxy(apiserver). 无法直接使用 tcp 。
func (c *TypedClient) PortForward(ctx context.Context, obj client.Object, port int) (*PortForwarder, error) {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return nil, err
	}

	if gvk.Kind != "Service" && gvk.Kind != "Pod" {
		return nil, fmt.Errorf("unsupported port forwarding of %s", gvk.GroupKind().String())
	}

	queries := url.Values{}
	queries.Set("port", strconv.Itoa(port))

	addr := fmt.Sprintf("%s/internal/core/v1/namespaces/%s/%s/%s/portforward?%s",
		c.serveraddr,
		obj.GetNamespace(),
		gvk.Kind,
		obj.GetName(),
		queries.Encode(),
	)
	forwarder, err := newPortForwarder(ctx, addr)
	if err != nil {
		return nil, err
	}
	return forwarder, nil
}

func (c *TypedClient) Proxy(ctx context.Context, obj client.Object, port int, req *http.Request, writer http.ResponseWriter, rewritefunc func(r *http.Response) error) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	if gvk.Kind != "Service" && gvk.Kind != "Pod" {
		return fmt.Errorf("unsupported proxy for %s", gvk.GroupKind().String())
	}

	addr := fmt.Sprintf("%s/internal/core/v1/namespaces/%s/%s/%s:%d/proxy",
		c.serveraddr,
		obj.GetNamespace(),
		gvk.Kind,
		obj.GetName(),
		port,
	)
	target, err := url.Parse(addr)
	if err != nil {
		return err
	}

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = target.Path + req.URL.Path
			req.Host = target.Host
		},
		Transport:      c.http.Transport,
		ModifyResponse: rewritefunc,
	}).ServeHTTP(writer, req)
	return nil
}

// ResponseBodyRewriter 会正确处理 gzip 以及 deflate 的content-encodeing 以及response 的content-length
// 用于需要修改代理的响应体是非常有用
func ResponseBodyRewriter(rewritefunc func(io.Reader, io.Writer) error) func(resp *http.Response) error {
	return func(r *http.Response) error {
		reader := r.Body
		writer := &bytes.Buffer{}

		defer func() {
			r.Body.Close()
			r.Body = io.NopCloser(writer)
			r.ContentLength = int64(writer.Len())
			r.Header.Set("Content-Length", strconv.Itoa(writer.Len()))
		}()

		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			gzr, err := gzip.NewReader(reader)
			if err != nil {
				return err
			}
			gzw := gzip.NewWriter(writer)
			defer func() {
				gzw.Close()
				gzw.Flush()
			}()
			return rewritefunc(gzr, gzw)
		case "deflate":
			flw, err := flate.NewWriter(writer, 0)
			if err != nil {
				return err
			}
			defer func() {
				flw.Close()
				flw.Flush()
			}()
			return rewritefunc(flate.NewReader(reader), flw)
		default:
			return rewritefunc(reader, writer)
		}
	}
}
