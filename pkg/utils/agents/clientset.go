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
	"k8s.io/client-go/tools/clientcmd"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/httpsigs"
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
	h.database.DB().Scopes(models.ClusterIsNotDeleted).Model(&cluster).Pluck("cluster_name", &ret)
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
	if err := h.database.DB().Scopes(models.ClusterIsNotDeleted).Where(cluster).Model(cluster).Pluck("cluster_name", &ret).Error; err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, errors.New("no manager cluster found")
	}
	managerclustername := ret[0]
	return h.ClientOf(ctx, managerclustername)
}

func (h *ClientSet) serverInfoOf(ctx context.Context, cluster *models.Cluster) (*ServerInfo, error) {
	serverinfo := &ServerInfo{}

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
	baseaddr, err := url.Parse(restconfig.Host + h.apiServerProxyPath(true))
	if err != nil {
		return nil, err
	}
	serverinfo.Addr = baseaddr
	serverinfo.CA = restconfig.TLSClientConfig.CAData

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

type ServerInfo struct {
	Addr     *url.URL `json:"addr,omitempty"` // addr with api path prefix
	CA       []byte   `json:"ca,omitempty"`
	AuthInfo AuthInfo `json:"authInfo,omitempty"`
}

func (s *ServerInfo) TLSConfig() (*tls.Config, error) {
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

type AuthInfo struct {
	ClientCertificate []byte `json:"clientCertificate,omitempty"`
	ClientKey         []byte `json:"clientKey,omitempty"`
	Token             string `json:"token,omitempty"`
	Username          string `json:"username,omitempty"`
	Password          string `json:"password,omitempty"`
}

func (auth *AuthInfo) IsEmpty() bool {
	return len(auth.ClientCertificate) == 0 && len(auth.ClientKey) == 0 && auth.Token == "" && auth.Username == "" && auth.Password == ""
}

func (auth *AuthInfo) Proxy(req *http.Request) (*url.URL, error) {
	if auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+auth.Token)
		return nil, nil
	}
	if _, _, exist := req.BasicAuth(); !exist && auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
		return nil, nil
	}
	return nil, nil
}

func (h *ClientSet) newClientMeta(ctx context.Context, name string) (*ClientMeta, error) {
	cluster := &models.Cluster{}
	if err := h.database.DB().Scopes(models.ClusterIsNotDeleted).First(&cluster, "cluster_name = ?", name).Error; err != nil {
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
		Name:             name,
		BaseAddr:         baseaddr,
		APIServerAddr:    apiserveraddr,
		APIServerVersion: cluster.Version,
		TLSConfig:        tlsconfig,
		Proxy:            proxy.Proxy,
	}
	return climeta, nil
}

func httpSigner(basepath string) func(req *http.Request) (*url.URL, error) {
	signer := httpsigs.GetSigner()
	return func(req *http.Request) (*url.URL, error) {
		signer.Sign(req, basepath)
		return nil, nil
	}
}

type ChainedProxy []func(*http.Request) (*url.URL, error)

func (pc ChainedProxy) Proxy(req *http.Request) (*url.URL, error) {
	var finalurl *url.URL
	for _, p := range pc {
		if p == nil {
			continue
		}
		url, err := p(req)
		if err != nil {
			return nil, err
		}
		if url != nil {
			finalurl = url
		}
	}
	return finalurl, nil
}
