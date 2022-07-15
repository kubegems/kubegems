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

package istio

import (
	"github.com/kiali/kiali/business"
	"github.com/kiali/kiali/business/checkers"
	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
	networking_v1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
)

// COPY FROM KIALI
func BuildKubernetesServiceList(namespace models.Namespace, svcs []core_v1.Service, pods []core_v1.Pod, deployments []apps_v1.Deployment, istioConfigList models.IstioConfigList) *models.ServiceList {
	services := make([]models.ServiceOverview, len(svcs))
	conf := config.Get()
	// Convert each k8s service into our model
	for i, item := range svcs {
		sPods := kubernetes.FilterPodsForService(&item, pods)
		/** Check if Service has istioSidecar deployed */
		mPods := models.Pods{}
		mPods.Parse(sPods)
		hasSidecar := mPods.HasAnyIstioSidecar()
		svcVirtualServices := kubernetes.FilterVirtualServices(istioConfigList.VirtualServices, item.Namespace, item.Name)
		svcDestinationRules := kubernetes.FilterDestinationRules(istioConfigList.DestinationRules, item.Namespace, item.Name)
		svcGateways := kubernetes.FilterGatewaysByVS(istioConfigList.Gateways, svcVirtualServices)
		svcReferences := make([]*models.IstioValidationKey, 0)
		for _, vs := range svcVirtualServices {
			ref := models.BuildKey(vs.Kind, vs.Name, vs.Namespace)
			svcReferences = append(svcReferences, &ref)
		}
		for _, vs := range svcDestinationRules {
			ref := models.BuildKey(vs.Kind, vs.Name, vs.Namespace)
			svcReferences = append(svcReferences, &ref)
		}
		for _, gw := range svcGateways {
			ref := models.BuildKey(gw.Kind, gw.Name, gw.Namespace)
			svcReferences = append(svcReferences, &ref)
		}
		svcReferences = business.FilterUniqueIstioReferences(svcReferences)

		kialiWizard := getVSKialiScenario(svcVirtualServices)
		if kialiWizard == "" {
			kialiWizard = getDRKialiScenario(svcDestinationRules)
		}

		/** Check if Service has the label app required by Istio */
		_, appLabel := item.Spec.Selector[conf.IstioLabels.AppLabelName]
		/** Check if Service has additional item icon */
		services[i] = models.ServiceOverview{
			Name:                   item.Name,
			Namespace:              item.Namespace,
			IstioSidecar:           hasSidecar,
			AppLabel:               appLabel,
			AdditionalDetailSample: models.GetFirstAdditionalIcon(conf, item.ObjectMeta.Annotations),
			HealthAnnotations:      models.GetHealthAnnotation(item.Annotations, models.GetHealthConfigAnnotation()),
			Labels:                 item.Labels,
			Selector:               item.Spec.Selector,
			IstioReferences:        svcReferences,
			KialiWizard:            kialiWizard,
		}
	}

	return &models.ServiceList{Namespace: namespace, Services: services}
}

func GetServiceValidations(services []core_v1.Service, deployments []apps_v1.Deployment, pods []core_v1.Pod) models.IstioValidations {
	validations := checkers.ServiceChecker{
		Services:    services,
		Deployments: deployments,
		Pods:        pods,
	}.Check()

	return validations
}

func getVSKialiScenario(vs []networking_v1alpha3.VirtualService) string {
	scenario := ""
	for _, v := range vs {
		if scenario, ok := v.Labels["kiali_wizard"]; ok {
			return scenario
		}
	}
	return scenario
}

func getDRKialiScenario(dr []networking_v1alpha3.DestinationRule) string {
	scenario := ""
	for _, d := range dr {
		if scenario, ok := d.Labels["kiali_wizard"]; ok {
			return scenario
		}
	}
	return scenario
}
