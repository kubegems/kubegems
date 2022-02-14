package agents

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"kubegems.io/pkg/models"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/httpsigs"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/system"
)

const (
	AgentModeApiServer = "apiServerProxy"
	AgentModeAHTTP     = "http"
	AgentModeHTTPS     = "https"
)

type ClientSet struct {
	options *system.SystemOptions
	databse *database.Database
	clients sync.Map // name -> *Client
}

func NewClientSet(databse *database.Database, options *system.SystemOptions) (*ClientSet, error) {
	return &ClientSet{
		databse: databse,
		options: options,
	}, nil
}

func (h *ClientSet) apiserverProxyPath() string {
	const apiServerProxyPrefix = "/api/v1/namespaces/%s/services/%s:%d/proxy"
	return fmt.Sprintf(apiServerProxyPrefix, h.options.AgentNamespace, h.options.AgentServiceName, h.options.AgentServicePort)
}

func (h *ClientSet) Clusters() []string {
	var (
		ret     []string
		cluster models.Cluster
	)
	h.databse.DB().Model(&cluster).Pluck("cluster_name", &ret)
	return ret
}

func (h *ClientSet) ClientOf(ctx context.Context, name string) (*WrappedClient, error) {
	if v, ok := h.clients.Load(name); ok {
		return v.(*WrappedClient), nil
	}
	if client, err := h.newclient(ctx, name); err != nil {
		return nil, err
	} else {
		// extend
		h.extendClients(client)

		h.clients.Store(name, client)
		return client, nil
	}
}

func (h *ClientSet) extendClients(cli *WrappedClient) {
	cli.HttpClient = NewHttpClientFrom(cli)
	cli.ProxyClient = NewProxyClientFrom(cli)
	cli.TypedClient = NewTypedClientFrom(cli)
}

func (h *ClientSet) ClientOfManager(ctx context.Context) (*WrappedClient, error) {
	ret := []string{}
	cluster := &models.Cluster{Primary: true}
	if err := h.databse.DB().Where(cluster).Model(cluster).Pluck("cluster_name", &ret).Error; err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, errors.New("no manager cluster found")
	}
	managerclustername := ret[0]
	return h.ClientOf(ctx, managerclustername)
}

func (h *ClientSet) newclient(ctx context.Context, name string) (*WrappedClient, error) {
	cluster := &models.Cluster{}
	if err := h.databse.DB().First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return nil, err
	}
	addr, mode, cert, key, ca, kubeconfig := cluster.AgentAddr, cluster.Mode,
		[]byte(cluster.AgentCert), []byte(cluster.AgentKey), []byte(cluster.AgentCA), []byte(cluster.KubeConfig)

	switch mode {
	case AgentModeApiServer:
		apiserver, kubecliCert, kubecliKey, kubeca, err := kube.GetKubeconfigInfos(kubeconfig)
		if err != nil {
			return nil, err
		}
		tlsconfig, err := tlsConfigFrom(kubecliCert, kubecliKey, kubeca)
		if err != nil {
			return nil, err
		}

		baseurl, err := url.Parse(apiserver + h.apiserverProxyPath())
		if err != nil {
			return nil, err
		}
		cli := &WrappedClient{
			Name:     name,
			BaseAddr: baseurl,
			Timeout:  time.Second * time.Duration(h.options.AgentTimeout),
			transport: &http.Transport{
				TLSClientConfig: tlsconfig,
				Proxy:           getRequestProxy(h.apiserverProxyPath()),
			},
		}
		return cli, nil
	case "", AgentModeHTTPS:
		tlsconfig, err := tlsConfigFrom(cert, key, ca)
		if err != nil {
			return nil, err
		}

		baseurl, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		cli := &WrappedClient{
			Name:     name,
			Timeout:  time.Second * time.Duration(h.options.AgentTimeout),
			BaseAddr: baseurl,
			transport: &http.Transport{
				TLSClientConfig: tlsconfig,
				Proxy:           getRequestProxy(path.Join("/v1/proxy/cluster", name)),
			},
		}
		return cli, nil
	case AgentModeAHTTP:
		baseurl, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		cli := &WrappedClient{
			Name:     name,
			Timeout:  time.Second * time.Duration(h.options.AgentTimeout),
			BaseAddr: baseurl,
			transport: &http.Transport{
				Proxy: getRequestProxy(path.Join("/v1/proxy/cluster", name)),
			},
		}
		return cli, nil
	default:
		return nil, fmt.Errorf("unsupported agent mode")
	}
}

func tlsConfigFrom(cert, key, ca []byte) (*tls.Config, error) {
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	caCertPool.AppendCertsFromPEM(ca)
	return &tls.Config{RootCAs: caCertPool, Certificates: []tls.Certificate{certificate}}, nil
}

func getRequestProxy(pathPrefix string) func(req *http.Request) (*url.URL, error) {
	signer := httpsigs.GetSigner()
	return func(req *http.Request) (*url.URL, error) {
		signer.Sign(req, pathPrefix)
		return nil, nil
	}
}
