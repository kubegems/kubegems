package clients

import (
	"io"
	"net/http"

	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type ClientsProxy struct {
	clients *agents.ClientSet
}

func NewClientsProxy(clients *agents.ClientSet) *ClientsProxy {
	return &ClientsProxy{
		clients: clients,
	}
}

// HandlerToCluster returns a http.Handler that proxies requests to the given cluster.
// Fixme: this is a temporary solution, we will use a better way to proxy requests in the future.
func (p *ClientsProxy) HandlerToCluster(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cli, err := p.clients.ClientOf(r.Context(), name)
		if err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		resp, err := cli.DoRawRequest(r.Context(), agents.Request{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.Query(),
			Headers: r.Header,
			Body:    r.Body,
		})
		if err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		for k, v := range resp.Header {
			w.Header().Set(k, v[0])
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
