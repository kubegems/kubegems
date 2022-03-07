package agents

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"k8s.io/client-go/rest"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/httpsigs"
	"kubegems.io/pkg/utils/kube"
)

type ClientSet struct {
	options  *Options
	database *database.Database
	clients  sync.Map // name -> *Client
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
	return &ClientSet{
		database: database,
		options:  NewDefaultOptions(), // default options,if override by config from database
	}, nil
}

func (h *ClientSet) apiServerProxyPath(isHttps bool) string {
	template := "/api/v1/namespaces/%s/services/%s:%d/proxy"
	if isHttps {
		template = "/api/v1/namespaces/%s/services/https:%s:%d/proxy"
	}
	return fmt.Sprintf(template, h.options.Namespace, h.options.ServiceName, h.options.ServicePort)
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
	if err := h.database.DB().Where(cluster).Model(cluster).Pluck("cluster_name", &ret).Error; err != nil {
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
	// always using https via apiserver proxy now
	cluster.AgentAddr = apiserver + h.apiServerProxyPath(true)
	cluster.AgentCert = string(kubecliCert)
	cluster.AgentKey = string(kubecliKey)
	cluster.AgentCA = string(kubeca)

	// update databse
	if err := h.database.DB().Save(cluster).Error; err != nil {
		return err
	}
	return nil
}

func (h *ClientSet) newClientMeta(ctx context.Context, name string) (*ClientMeta, error) {
	cluster := &models.Cluster{}
	if err := h.database.DB().First(&cluster, "cluster_name = ?", name).Error; err != nil {
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
	apiserveraddr, err := url.Parse(cluster.APIServer)
	if err != nil {
		return nil, err
	}
	climeta := &ClientMeta{
		Name:             name,
		BaseAddr:         baseaddr,
		APIServerAddr:    apiserveraddr,
		APIServerVersion: cluster.Version,
		Signer:           getRequestProxy(h.apiServerProxyPath(true)),
	}
	defaultRoudTripper := &http.Transport{
		Proxy: climeta.Signer,
	}

	if baseaddr.Scheme == "https" {
		if cluster.AgentCert != "" && cluster.AgentKey != "" {
			cert, key, ca := []byte(cluster.AgentCert), []byte(cluster.AgentKey), []byte(cluster.AgentCA)
			tlsconfig, err := tlsConfigFrom(cert, key, ca)
			if err != nil {
				return nil, err
			}
			climeta.TlsConfig = tlsconfig
			defaultRoudTripper.TLSClientConfig = tlsconfig
			climeta.Transport = defaultRoudTripper
		} else {
			cfg, err := kube.GetKubeRestConfig(cluster.KubeConfig)
			if err != nil {
				return nil, err
			}
			tlsCfg, err := rest.TLSConfigFor(cfg)
			if err != nil {
				return nil, err
			}
			climeta.TlsConfig = tlsCfg
			climeta.Restconfig = cfg
			defaultRoudTripper.TLSClientConfig = tlsCfg
			rt, err := rest.HTTPWrappersForConfig(cfg, defaultRoudTripper)
			if err != nil {
				return nil, err
			}
			climeta.Transport = rt
		}
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
