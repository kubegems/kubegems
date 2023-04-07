package task

import (
	"fmt"
	"strings"
	"sync"

	edgeclient "kubegems.io/kubegems/pkg/edge/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EdgeClientsHolder struct {
	EdgeServerAddr string
	clients        sync.Map
}

func NewEdgeClientsHolder(edgeServerAddr string) (*EdgeClientsHolder, error) {
	if !strings.HasPrefix(edgeServerAddr, "http://") && !strings.HasPrefix(edgeServerAddr, "https://") {
		return nil, fmt.Errorf("scheme is required in edge server address")
	}
	return &EdgeClientsHolder{EdgeServerAddr: edgeServerAddr}, nil
}

func (c *EdgeClientsHolder) Get(uid string) (client.Client, error) {
	if cli, ok := c.clients.Load(uid); ok {
		// nolint: forcetypeassert
		return cli.(client.Client), nil
	}
	cli, err := edgeclient.NewEdgeClient(c.EdgeServerAddr, uid)
	if err != nil {
		return nil, err
	}
	c.clients.Store(uid, cli)
	return cli, nil
}
