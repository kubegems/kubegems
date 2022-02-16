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

	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/httpsigs"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/system"
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
	cli := newClient(*meta)

	h.clients.Store(name, cli)
	return cli, nil
}

func (h *ClientSet) ClientOfManager(ctx context.Context) (Client, error) {
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

func (h *ClientSet) newClientMeta(ctx context.Context, name string) (*ClientMeta, error) {
	cluster := &models.Cluster{}
	if err := h.databse.DB().First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return nil, err
	}
	addr, mode := cluster.AgentAddr, cluster.Mode

	cert, key, ca, kubeconfig := []byte(cluster.AgentCert), []byte(cluster.AgentKey), []byte(cluster.AgentCA), []byte(cluster.KubeConfig)

	switch mode {
	case AgentModeApiServer, "apiserver":
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
		cli := &ClientMeta{
			Name:     name,
			BaseAddr: baseurl,
			Transport: &http.Transport{
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
		cli := &ClientMeta{
			Name:     name,
			BaseAddr: baseurl,
			Transport: &http.Transport{
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
		cli := &ClientMeta{
			Name:     name,
			BaseAddr: baseurl,
			Transport: &http.Transport{
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
