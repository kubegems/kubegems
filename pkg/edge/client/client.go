package client

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEdgeClient creates a new EdgeClient.
func NewEdgeClient(edgeServerAddr string, uid string) (client.Client, error) {
	if uid == "" {
		return nil, fmt.Errorf("device id is empty")
	}
	u, err := url.Parse(fmt.Sprintf("%s/v1/edge-clusters/%s/proxy", edgeServerAddr, uid))
	if err != nil {
		return nil, err
	}
	clioptions := &agents.ClientOptions{
		Addr: u,
		TLS:  &tls.Config{InsecureSkipVerify: true},
	}
	cli := agents.NewTypedClient(clioptions, kube.GetScheme())
	return cli, nil
}
