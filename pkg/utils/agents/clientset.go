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
)

type ClientSet struct {
	options *Options
	databse *database.Database
	clients sync.Map // name -> *Client
}

func NewClientSet(databse *database.Database) (*ClientSet, error) {
	return &ClientSet{
		databse: databse,
		options: NewDefaultOptions(), // default options,if override by config from database
	}, nil
}

func (h *ClientSet) apiServerProxyPath() string {
	return fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy",
		h.options.Namespace, h.options.ServiceName, h.options.ServicePort)
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

func (h *ClientSet) completeFromKubeconfig(ctx context.Context, cluster *models.Cluster) error {
	kubeconfig := []byte(cluster.KubeConfig)
	apiserver, kubecliCert, kubecliKey, kubeca, err := kube.GetKubeconfigInfos(kubeconfig)
	if err != nil {
		return err
	}
	cluster.AgentAddr = apiserver + h.apiServerProxyPath()
	cluster.AgentCert = string(kubecliCert)
	cluster.AgentKey = string(kubecliKey)
	cluster.AgentCA = string(kubeca)

	// update databse
	if err := h.databse.DB().Save(cluster).Error; err != nil {
		return err
	}
	return nil
}

func (h *ClientSet) newClientMeta(ctx context.Context, name string) (*ClientMeta, error) {
	cluster := &models.Cluster{}
	if err := h.databse.DB().First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return nil, err
	}

	// 如果 agent addr 为空，则使用 apiserver 模式回填
	if len(cluster.AgentAddr) == 0 {
		if err := h.completeFromKubeconfig(ctx, cluster); err != nil {
			return nil, err
		}
	}

	baseaddr, err := url.Parse(cluster.AgentAddr)
	if err != nil {
		return nil, err
	}

	climeta := &ClientMeta{
		Name:     name,
		BaseAddr: baseaddr,
		Transport: &http.Transport{
			Proxy: getRequestProxy(path.Join("/v1/proxy/cluster", name)),
		},
	}

	if baseaddr.Scheme == "https" {
		cert, key, ca := []byte(cluster.AgentCert), []byte(cluster.AgentKey), []byte(cluster.AgentCA)
		tlsconfig, err := tlsConfigFrom(cert, key, ca)
		if err != nil {
			return nil, err
		}
		climeta.Transport.TLSClientConfig = tlsconfig
	}
	return climeta, nil
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
