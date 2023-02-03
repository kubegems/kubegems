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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"sync"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
)

type ClientSet struct {
	database *database.Database
	clients  sync.Map // name -> *Client
	tracer   trace.Tracer
}

// Initialize for gorm plugin
func (h *ClientSet) Initialize(db *gorm.DB) error {
	return nil
}

// Name for gorm plugin
func (h *ClientSet) Name() string {
	return "agentcli"
}

func NewClientSet(database *database.Database) (*ClientSet, error) {
	return &ClientSet{database: database, tracer: otel.GetTracerProvider().Tracer("kubegems.io/kubegems")}, nil
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

func (h *ClientSet) Clusters() []string {
	var (
		ret     []string
		cluster models.Cluster
	)
	h.database.DB().Model(&cluster).Pluck("cluster_name", &ret)
	return ret
}

// ExecuteInEachCluster Execute in each cluster concurrently
func (h ClientSet) ExecuteInEachCluster(ctx context.Context, f func(ctx context.Context, cli Client) error) error {
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

	meta, err := h.newClientMeta(ctx, name)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(meta.ServerInfo.RestConfig)
	if err != nil {
		return nil, err
	}
	cli := newClient(*meta, clientset, h.tracer)
	h.clients.Store(name, cli)
	return cli, nil
}

func (h *ClientSet) serverInfoOf(ctx context.Context, cluster *models.Cluster) (*serverInfo, error) {
	serverinfo := &serverInfo{}

	// from origin
	if len(cluster.KubeConfig) == 0 || cluster.AgentAddr != "" {
		baseaddr, err := url.Parse(cluster.AgentAddr)
		if err != nil {
			return nil, err
		}
		serverinfo.Addr = baseaddr
		serverinfo.CA = []byte(cluster.AgentCA)

		serverinfo.AuthInfo.ClientCertificate = []byte(cluster.AgentCert)
		serverinfo.AuthInfo.ClientKey = []byte(cluster.AgentKey)

		return serverinfo, nil
	}

	// from kubeconfig
	kubeconfig := []byte(cluster.KubeConfig)
	restconfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	cluster.APIServer = restconfig.Host

	// complete server info
	path := ApiServerProxyPath(cluster.InstallNamespace, "https", "", "")
	baseaddr, err := url.Parse(restconfig.Host + path)
	if err != nil {
		return nil, err
	}
	serverinfo.Addr = baseaddr
	serverinfo.CA = restconfig.TLSClientConfig.CAData
	serverinfo.RestConfig = restconfig

	// complete auth info
	if authinfo := &serverinfo.AuthInfo; authinfo.IsEmpty() {
		transportconfig, err := restconfig.TransportConfig()
		if err != nil {
			return nil, err
		}
		switch {
		case transportconfig.HasBasicAuth():
			authinfo.Username = transportconfig.Username
			authinfo.Password = transportconfig.Password
		case transportconfig.HasTokenAuth():
			authinfo.Token = transportconfig.BearerToken
		case transportconfig.HasCertAuth():
			authinfo.ClientCertificate = transportconfig.TLS.CertData
			authinfo.ClientKey = transportconfig.TLS.KeyData
		}
	}
	return serverinfo, nil
}

type serverInfo struct {
	Addr       *url.URL
	CA         []byte
	AuthInfo   AuthInfo
	RestConfig *rest.Config
}

func (s *serverInfo) TLSConfig() (*tls.Config, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		caCertPool = x509.NewCertPool()
	}
	if s.CA != nil {
		caCertPool.AppendCertsFromPEM(s.CA)
	}
	tlsconfig := &tls.Config{RootCAs: caCertPool}
	cert, key := s.AuthInfo.ClientCertificate, s.AuthInfo.ClientKey
	if len(cert) > 0 && len(key) > 0 {
		certificate, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		tlsconfig.Certificates = append(tlsconfig.Certificates, certificate)
	}
	return tlsconfig, nil
}

func (h *ClientSet) newClientMeta(ctx context.Context, name string) (*ClientMeta, error) {
	cluster := &models.Cluster{}
	if err := h.database.DB().WithContext(ctx).First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return nil, err
	}

	serverinfo, err := h.serverInfoOf(ctx, cluster)
	if err != nil {
		return nil, err
	}
	baseaddr := serverinfo.Addr

	// TODO: consider replace with baseaddr
	apiserveraddr, err := url.Parse(cluster.APIServer)
	if err != nil {
		return nil, err
	}

	proxy := ChainedProxy{
		httpSigner(baseaddr.Path), // http sig
		serverinfo.AuthInfo.Proxy, // basic auth / token auth
	}

	// tls
	tlsconfig, err := serverinfo.TLSConfig()
	if err != nil {
		return nil, err
	}

	climeta := &ClientMeta{
		Name:          name,
		BaseAddr:      baseaddr,
		APIServerAddr: apiserveraddr,
		TLSConfig:     tlsconfig,
		ServerInfo:    *serverinfo,
		Proxy:         proxy.Proxy,
	}
	return climeta, nil
}
