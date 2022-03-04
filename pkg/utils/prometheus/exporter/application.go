package exporter

import (
	"context"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/pkg/apis/application"
	gemlabels "kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/argo"
)

type ApplicationCollector struct {
	projectInfo *prometheus.Desc

	*argo.Client
	mutex sync.Mutex
}

func NewApplicationCollector(cli *argo.Client) func(_ *log.Logger) (Collector, error) {
	return func(_ *log.Logger) (Collector, error) {
		return &ApplicationCollector{
			projectInfo: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "application", "status"),
				"Gems application status",
				[]string{"application_name", "creator", "from", "environment_name", "project_name", "tenant_name", "cluster_name", "namespace", "status"},
				nil,
			),
			Client: cli,
		}, nil
	}
}

func (c *ApplicationCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	apps, err := c.ListArgoApp(context.TODO(), labels.Everything())
	if err != nil {
		log.Errorf("faild to list apps: %v", err)
		return err
	}

	for _, v := range apps.Items {
		if v.Labels != nil && v.Labels[gemlabels.LabelApplication] != "" {
			ch <- prometheus.MustNewConstMetric(
				c.projectInfo,
				prometheus.GaugeValue,
				1,

				v.Labels[gemlabels.LabelApplication],
				v.Annotations[application.AnnotationCreator],
				v.Labels[application.LabelFrom],
				v.Labels[gemlabels.LabelEnvironment],
				v.Labels[gemlabels.LabelProject],
				v.Labels[gemlabels.LabelTenant],
				strings.TrimPrefix(v.Spec.Destination.Name, "argocd-cluster-"),
				v.Spec.Destination.Namespace,
				string(v.Status.Health.Status),
			)
		}
	}

	return nil
}
