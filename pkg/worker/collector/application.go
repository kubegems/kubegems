package collector

import (
	"context"
	"strings"
	"sync"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/prometheus/client_golang/prometheus"
	kubegemsapp "kubegems.io/pkg/apis/application"
	gemlabels "kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
)

var (
	argoAppCli application.ApplicationServiceClient
	dbinstance *database.Database
)

func Init(argo *argo.Client, db *database.Database) {
	_, argoAppCli = argo.ArgoCDcli.NewApplicationClientOrDie()
	dbinstance = db
}

type ApplicationCollector struct {
	projectInfo *prometheus.Desc
	mutex       sync.Mutex
}

func NewApplicationCollector() func(_ *log.Logger) (exporter.Collector, error) {
	return func(_ *log.Logger) (exporter.Collector, error) {
		return &ApplicationCollector{
			projectInfo: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetNamespace(), "application", "status"),
				"Gems application status",
				[]string{"application_name", "creator", "from", "environment_name", "project_name", "tenant_name", "cluster_name", "namespace", "status"},
				nil,
			),
		}, nil
	}
}

func (c *ApplicationCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	apps, err := argoAppCli.List(context.TODO(), &application.ApplicationQuery{})
	if err != nil {
		log.Errorf("faild to list apps: %v", err)
		return err
	}

	for _, v := range apps.Items {
		env := v.Labels[gemlabels.LabelEnvironment]
		proj := v.Labels[gemlabels.LabelProject]
		tenant := v.Labels[gemlabels.LabelTenant]
		ch <- prometheus.MustNewConstMetric(
			c.projectInfo,
			prometheus.GaugeValue,
			1,

			v.Labels[gemlabels.LabelApplication],
			v.Labels[kubegemsapp.AnnotationCreator],
			v.Labels[kubegemsapp.LabelFrom],
			env,
			proj,
			tenant,
			strings.TrimPrefix(v.Spec.Destination.Name, "argocd-cluster-"),
			v.Spec.Destination.Namespace,
			string(v.Status.Health.Status),
		)
	}

	return nil
}
