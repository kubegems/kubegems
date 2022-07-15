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

package webhooks

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	nginx_v1alpha1 "github.com/nginxinc/nginx-ingress-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/apis/networking"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func patchResoucesIfNotExists(containers []corev1.Container, limits, requests corev1.ResourceList) {
	for idx, container := range containers {
		if _, exist := container.Resources.Limits[corev1.ResourceCPU]; !exist {
			containers[idx].Resources.Limits[corev1.ResourceCPU] = limits.Cpu().DeepCopy()
		}
		if _, exist := container.Resources.Limits[corev1.ResourceMemory]; !exist {
			containers[idx].Resources.Limits[corev1.ResourceMemory] = limits.Memory().DeepCopy()
		}
		if _, exist := container.Resources.Requests[corev1.ResourceCPU]; !exist {
			containers[idx].Resources.Requests[corev1.ResourceCPU] = requests.Cpu().DeepCopy()
		}
		if _, exist := container.Resources.Requests[corev1.ResourceMemory]; !exist {
			containers[idx].Resources.Requests[corev1.ResourceMemory] = requests.Memory().DeepCopy()
		}
	}
}

func patchEpherContainerResoucesIfNotExists(containers []corev1.EphemeralContainer, limits, requests corev1.ResourceList) {
	for idx, container := range containers {
		if _, exist := container.Resources.Limits[corev1.ResourceCPU]; !exist {
			containers[idx].Resources.Limits[corev1.ResourceCPU] = limits.Cpu().DeepCopy()
		}
		if _, exist := container.Resources.Limits[corev1.ResourceMemory]; !exist {
			containers[idx].Resources.Limits[corev1.ResourceMemory] = limits.Memory().DeepCopy()
		}
		if _, exist := container.Resources.Requests[corev1.ResourceCPU]; !exist {
			containers[idx].Resources.Requests[corev1.ResourceCPU] = requests.Cpu().DeepCopy()
		}
		if _, exist := container.Resources.Requests[corev1.ResourceMemory]; !exist {
			containers[idx].Resources.Requests[corev1.ResourceMemory] = requests.Memory().DeepCopy()
		}
	}
}

func (r *ResourceValidate) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Kind {
	case gkvTenant:
		return r.ValidateTenant(ctx, req)
	case gkvTenantResourceQuota:
		return r.ValidateTenantResourceQuota(ctx, req)
	case gkvTenantGateway:
		return r.ValidateTenantGateway(ctx, req)
	case gkvTenantNetworkPolicy:
		return r.ValidateTenantNetworkPolicy(ctx, req)
	case gkvEnvironment:
		return r.ValidateEnvironment(ctx, req)
	case gkvNamespace:
		return r.ValidateNamespace(ctx, req)
	case gvkIstioGateway:
		return r.ValidateIstioGateway(ctx, req)
	default:
		return admission.Allowed("pass")
	}
}

func (r *ResourceMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Kind {
	case gkvEnvironment:
		return r.MutateEnvironment(ctx, req)
	case gkvTenantResourceQuota:
		return r.MutateTenantResourceQuota(ctx, req)
	case gkvTenantGateway:
		return r.MutateTenantGateway(ctx, req)
	case gkvIngress:
		return r.MutateIngress(ctx, req)
	default:
		return admission.Allowed("pass")
	}
}

func (r *LabelInjectorMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	return r.CommonInjectLabel(ctx, req)
}

type ResourceValidate struct {
	Client  client.Client
	decoder *admission.Decoder
	Log     logr.Logger
}

func (r *ResourceValidate) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

type ResourceMutate struct {
	Client  client.Client
	decoder *admission.Decoder
	Log     logr.Logger
	Repo    string
}

type LabelInjectorMutate struct {
	Client  client.Client
	decoder *admission.Decoder
	Log     logr.Logger
}

func (r *LabelInjectorMutate) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

func (r *ResourceMutate) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

func GetLabelInjectorMutateHandler(client *client.Client, log *logr.Logger) *webhook.Admission {
	return &webhook.Admission{Handler: &LabelInjectorMutate{Client: *client, Log: *log}}
}

func GetMutateHandler(client *client.Client, log *logr.Logger, repo string) *webhook.Admission {
	return &webhook.Admission{Handler: &ResourceMutate{Client: *client, Log: *log, Repo: repo}}
}

func GetValidateHandler(client *client.Client, log *logr.Logger) *webhook.Admission {
	return &webhook.Admission{Handler: &ResourceValidate{Client: *client, Log: *log}}
}

func CreateDefaultTenantGateway(client client.Client, log logr.Logger) {
	tg := gemsv1beta1.TenantGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultGatewayName,
			Namespace: gemlabels.NamespaceGateway,
			Labels: map[string]string{
				networking.LabelIngressClass: defaultGatewayName,
			},
		},
		Spec: gemsv1beta1.TenantGatewaySpec{
			Tenant:       notenant,
			IngressClass: defaultGatewayName,
		},
	}

	execute := func() error {
		nicList := nginx_v1alpha1.NginxIngressControllerList{}
		if err := client.List(context.TODO(), &nicList); err != nil {
			return err
		}
		return client.Create(context.TODO(), &tg)
	}

	for {
		err := execute()
		switch {
		case err == nil:
			log.Info("succeed to create default tenant gateway")
			return
		case apierrors.IsAlreadyExists(err):
			log.Info("default tenant gateway already exist")
			return
		default:
			log.Info(fmt.Sprintf("failed to create default tenant gateway: %v, waiting to try again", err))
			time.Sleep(10 * time.Second)
		}
	}
}
