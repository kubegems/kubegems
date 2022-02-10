package proxyutil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/sets"
	"kubegems.io/pkg/log"
)

/*
代理后修改reponse中的绝对地址
https://github1s.com/kubernetes/apimachinery/blob/master/pkg/util/proxy/transport.go
*/

const apiServerProxyPrefix = "/api/v1/namespaces/gemcloud-system/services/gems-agent:8041/proxy"

// Transport is a transport for text/html content that replaces URLs in html
// content with the prefix of the proxy server
type Transport struct {
	Scheme      string
	Host        string
	PathPrepend string
	AgentPrefix string

	http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add reverse proxy headers.
	forwardedURI := path.Join(t.PathPrepend, req.URL.Path)
	if strings.HasSuffix(req.URL.Path, "/") {
		forwardedURI = forwardedURI + "/"
	}
	req.Header.Set("X-Forwarded-Uri", forwardedURI)
	if len(t.Host) > 0 {
		req.Header.Set("X-Forwarded-Host", t.Host)
	}
	if len(t.Scheme) > 0 {
		req.Header.Set("X-Forwarded-Proto", t.Scheme)
	}

	rt := t.RoundTripper
	if rt == nil {
		rt = http.DefaultTransport
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return nil, errors.NewServiceUnavailable(fmt.Sprintf("error trying to reach service: %v", err))
	}

	if redirect := resp.Header.Get("Location"); redirect != "" {
		resp.Header.Set("Location", t.rewriteURL(redirect, req.URL, req.Host))
		return resp, nil
	}

	cType := resp.Header.Get("Content-Type")
	cType = strings.TrimSpace(strings.SplitN(cType, ";", 2)[0])
	if cType != "text/html" {
		// Do nothing, simply pass through
		return resp, nil
	}

	return t.rewriteResponse(req, resp)
}

var _ = net.RoundTripperWrapper(&Transport{})

func (rt *Transport) WrappedRoundTripper() http.RoundTripper {
	return rt.RoundTripper
}

// rewriteURL rewrites a single URL to go through the proxy, if the URL refers
// to the same host as sourceURL, which is the page on which the target URL
// occurred, or if the URL matches the sourceRequestHost. If any error occurs (e.g.
// parsing), it returns targetURL.
func (t *Transport) rewriteURL(targetURL string, sourceURL *url.URL, sourceRequestHost string) string {
	url, err := url.Parse(targetURL)
	if err != nil {
		return targetURL
	}

	// Example:
	//      When API server processes a proxy request to a service (e.g. /api/v1/namespace/foo/service/bar/proxy/),
	//      the sourceURL.Host (i.e. req.URL.Host) is the endpoint IP address of the service. The
	//      sourceRequestHost (i.e. req.Host) is the Host header that specifies the host on which the
	//      URL is sought, which can be different from sourceURL.Host. For example, if user sends the
	//      request through "kubectl proxy" locally (i.e. localhost:8001/api/v1/namespace/foo/service/bar/proxy/),
	//      sourceRequestHost is "localhost:8001".
	//
	//      If the service's response URL contains non-empty host, and url.Host is equal to either sourceURL.Host
	//      or sourceRequestHost, we should not consider the returned URL to be a completely different host.
	//      It's the API server's responsibility to rewrite a same-host-and-absolute-path URL and append the
	//      necessary URL prefix (i.e. /api/v1/namespace/foo/service/bar/proxy/).
	isDifferentHost := url.Host != "" && url.Host != sourceURL.Host && url.Host != sourceRequestHost
	isRelative := !strings.HasPrefix(url.Path, "/")
	if isDifferentHost || isRelative {
		return targetURL
	}

	// Do not rewrite scheme and host if the Transport has empty scheme and host
	// when targetURL already contains the sourceRequestHost
	if !(url.Host == sourceRequestHost && t.Scheme == "" && t.Host == "") {
		url.Scheme = t.Scheme
		url.Host = t.Host
	}

	origPath := url.Path
	// Do not rewrite URL if the sourceURL already contains the necessary prefix.
	if strings.HasPrefix(url.Path, t.PathPrepend) {
		return url.String()
	}
	url.Path = path.Join(t.PathPrepend, url.Path)
	if strings.HasSuffix(origPath, "/") {
		// Add back the trailing slash, which was stripped by path.Join().
		url.Path += "/"
	}

	return url.String()
}

// rewriteResponse modifies an HTML response by updating absolute links referring
// to the original host to instead refer to the proxy transport.
func (t *Transport) rewriteResponse(req *http.Request, resp *http.Response) (*http.Response, error) {
	origBody := resp.Body
	defer origBody.Close()

	newContent := &bytes.Buffer{}
	var reader io.Reader = origBody
	var writer io.Writer = newContent
	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("errorf making gzip reader: %v", err)
		}
		gzw := gzip.NewWriter(writer)
		defer gzw.Close()
		writer = gzw
	case "deflate":
		var err error
		reader = flate.NewReader(reader)
		flw, err := flate.NewWriter(writer, flate.BestCompression)
		if err != nil {
			return nil, fmt.Errorf("errorf making flate writer: %v", err)
		}
		defer func() {
			flw.Close()
			flw.Flush()
		}()
		writer = flw
	case "":
		// This is fine
	default:
		// Some encoding we don't understand-- don't try to parse this
		log.Errorf("Proxy encountered encoding %v for text/html; can't understand this so not fixing links.", encoding)
		return resp, nil
	}

	urlRewriter := func(targetUrl string) string {
		targetUrl = strings.ReplaceAll(targetUrl, apiServerProxyPrefix, "")
		if t.AgentPrefix != "" {
			targetUrl = strings.ReplaceAll(targetUrl, t.AgentPrefix, "")
		}
		return t.rewriteURL(targetUrl, req.URL, req.Host)
	}
	err := rewriteHTML(reader, writer, urlRewriter)
	if err != nil {
		log.Errorf("Failed to rewrite URLs: %v", err)
		return resp, err
	}

	resp.Body = ioutil.NopCloser(newContent)
	// Update header node with new content-length
	// TODO: Remove any hash/signature headers here?
	resp.Header.Del("Content-Length")
	resp.ContentLength = int64(newContent.Len())

	return resp, err
}

// rewriteHTML scans the HTML for tags with url-valued attributes, and updates
// those values with the urlRewriter function. The updated HTML is output to the
// writer.
func rewriteHTML(reader io.Reader, writer io.Writer, urlRewriter func(string) string) error {
	// Note: This assumes the content is UTF-8.
	tokenizer := html.NewTokenizer(reader)

	var err error
	for err == nil {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			err = tokenizer.Err()
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if urlAttrs, ok := atomsToAttrs[token.DataAtom]; ok {
				for i, attr := range token.Attr {
					if urlAttrs.Has(attr.Key) {
						token.Attr[i].Val = urlRewriter(attr.Val)
					}
				}
			}
			_, err = writer.Write([]byte(token.String()))
		default:
			_, err = writer.Write(tokenizer.Raw())
		}
	}
	if err != io.EOF {
		return err
	}
	return nil
}

var atomsToAttrs = map[atom.Atom]sets.String{
	atom.A:          sets.NewString("href"),
	atom.Applet:     sets.NewString("codebase"),
	atom.Area:       sets.NewString("href"),
	atom.Audio:      sets.NewString("src"),
	atom.Base:       sets.NewString("href"),
	atom.Blockquote: sets.NewString("cite"),
	atom.Body:       sets.NewString("background"),
	atom.Button:     sets.NewString("formaction"),
	atom.Command:    sets.NewString("icon"),
	atom.Del:        sets.NewString("cite"),
	atom.Embed:      sets.NewString("src"),
	atom.Form:       sets.NewString("action"),
	atom.Frame:      sets.NewString("longdesc", "src"),
	atom.Head:       sets.NewString("profile"),
	atom.Html:       sets.NewString("manifest"),
	atom.Iframe:     sets.NewString("longdesc", "src"),
	atom.Img:        sets.NewString("longdesc", "src", "usemap"),
	atom.Input:      sets.NewString("src", "usemap", "formaction"),
	atom.Ins:        sets.NewString("cite"),
	atom.Link:       sets.NewString("href"),
	atom.Object:     sets.NewString("classid", "codebase", "data", "usemap"),
	atom.Q:          sets.NewString("cite"),
	atom.Script:     sets.NewString("src"),
	atom.Source:     sets.NewString("src"),
	atom.Video:      sets.NewString("poster", "src"),

	// TODO: css URLs hidden in style elements.
}
