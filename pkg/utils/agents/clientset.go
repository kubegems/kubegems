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
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents/client"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/kube/schema"
)

type ClientSet struct {
	database *database.Database
	clients  sync.Map // name -> *Client
	tracer   trace.Tracer
}

func NewClientSet(database *database.Database) (*ClientSet, error) {
	return &ClientSet{database: database, tracer: otel.GetTracerProvider().Tracer("kubegems.io/kubegems")}, nil
}

func (h *ClientSet) Clusters() []string {
	var (
		ret     []string
		cluster models.Cluster
	)
	h.database.DB().Model(&cluster).Pluck("cluster_name", &ret)
	return ret
}

// ExecuteInEachCluster Execute in each cluster concurrently
func (h *ClientSet) ExecuteInEachCluster(ctx context.Context, f func(ctx context.Context, cli Client) error) error {
	g := errgroup.Group{}
	for _, v := range h.Clusters() {
		clustername := v
		g.Go(func() error {
			client, err := h.ClientOf(ctx, clustername)
			if err != nil {
				return err
			}

			return f(ctx, client)
		})
	}
	return g.Wait()
}

func (h *ClientSet) ClientOfManager(ctx context.Context) (Client, error) {
	ret := []string{}
	cluster := &models.Cluster{Primary: true}
	if err := h.database.DB().WithContext(ctx).Where(cluster).Model(cluster).Pluck("cluster_name", &ret).Error; err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, errors.New("no manager cluster found")
	}
	managerclustername := ret[0]
	return h.ClientOf(ctx, managerclustername)
}

// Invalidate a client of name cluster and recreate after.
func (h *ClientSet) Invalidate(ctx context.Context, name string) {
	h.clients.Delete(name)
}

func (h *ClientSet) ClientOf(ctx context.Context, name string) (Client, error) {
	if v, ok := h.clients.Load(name); ok {
		if cli, ok := v.(Client); ok {
			return cli, nil
		}
		return nil, fmt.Errorf("invalid client type: %T", v)
	}

	cli, err := h.newClientFor(ctx, name)
	if err != nil {
		return nil, err
	}

	h.clients.Store(name, cli)
	return cli, nil
}

func (h *ClientSet) newClientFor(ctx context.Context, name string) (Client, error) {
	clientOptions, kubeconfig, err := h.ClientOptionsOf(ctx, name)
	if err != nil {
		return nil, err
	}
	return NewDelegateClientClient(name, clientOptions, kubeconfig, schema.GetScheme(), h.tracer)
}

func (h *ClientSet) ClientOptionsOf(ctx context.Context, name string) (*client.Config, *rest.Config, error) {
	cluster := &models.Cluster{}
	if err := h.database.DB().WithContext(ctx).First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return nil, nil, err
	}
	// from origin
	if len(cluster.KubeConfig) == 0 || cluster.AgentAddr != "" {
		baseaddr, err := url.Parse(cluster.AgentAddr)
		if err != nil {
			return nil, nil, err
		}
		tlscfg, err := client.TLSConfigFrom([]byte(cluster.AgentCA), []byte(cluster.AgentCert), []byte(cluster.AgentKey))
		if err != nil {
			return nil, nil, err
		}
		info := &client.Config{
			Addr: baseaddr,
			TLS:  tlscfg,
		}
		return info, nil, nil
	}
	// from kubeconfig
	restconfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		return nil, nil, err
	}
	// use apiserver proxy to access agent
	baseaddr, err := url.Parse(restconfig.Host + ApiServerProxyPath(cluster.InstallNamespace, "https", "", ""))
	if err != nil {
		return nil, nil, err
	}
	tlscfg, err := client.TLSConfigFrom(restconfig.TLSClientConfig.CAData, restconfig.TLSClientConfig.CertData, restconfig.TLSClientConfig.KeyData)
	if err != nil {
		return nil, nil, err
	}
	serverinfo := &client.Config{
		Addr: baseaddr,
		TLS:  tlscfg,
		Auth: client.Auth{
			Token:    restconfig.BearerToken,
			Username: restconfig.Username,
			Password: restconfig.Password,
		},
	}
	return serverinfo, restconfig, nil
}

func ApiServerProxyPath(namespace, schema, svcname, port string) string {
	if namespace == "" {
		namespace = "kubegems-local"
	}
	if svcname == "" {
		svcname = "kubegems-local-agent"
	}
	if port == "" {
		port = "http" // include https
	}
	if schema != "" {
		template := "/api/v1/namespaces/%s/services/%s:%s:%s/proxy"
		return fmt.Sprintf(template, namespace, schema, svcname, port)
	} else {
		template := "/api/v1/namespaces/%s/services/%s:%s/proxy"
		return fmt.Sprintf(template, namespace, svcname, port)
	}
}
