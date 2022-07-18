package plugins

import (
	"github.com/emicklei/go-restful/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/route"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PluginsAPI struct {
	cli client.Client
}

type PluginsStatus struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func NewPluginsAPI() (*PluginsAPI, error) {
	// check 'this' cluster svc kubegems-models-store
	var cli client.Client
	if cfg, _ := kube.AutoClientConfig(); cfg != nil {
		c, err := client.New(cfg, client.Options{})
		if err != nil {
			return nil, err
		}
		cli = c
	} else {
		cli = kube.NoopClient{}
	}
	return &PluginsAPI{cli: cli}, nil
}

func (p *PluginsAPI) List(req *restful.Request, resp *restful.Response) {
	ret := []PluginsStatus{}

	// kubegems
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubegems-models-store",
			Namespace: "kubegems",
		},
	}
	if err := p.cli.Get(req.Request.Context(), client.ObjectKeyFromObject(svc), svc); err != nil {
		if errors.IsNotFound(err) {
			ret = append(ret, PluginsStatus{Name: "kubegems-models", Enabled: false})
		} else {
			response.Error(resp, err)
			return
		}
	} else {
		ret = append(ret, PluginsStatus{Name: "kubegems-models", Enabled: true})
	}
	response.OK(resp, ret)
}

func (p *PluginsAPI) RegisterRoute(rg *route.Group) {
	rg.
		Tag("plugins").
		AddRoutes(
			route.GET("/plugins").To(p.List).Doc("List plugins").Response([]PluginsStatus{}),
		)
}
