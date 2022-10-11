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

package deployment

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	machinelearningv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	gemsv1beta1 "kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const SeldonModelServeKind = "seldon"

type SeldonModelServe struct {
	Client        client.Client
	IngressHost   string
	IngressScheme string
}

func (r *SeldonModelServe) Watches() client.Object {
	return &machinelearningv1.SeldonDeployment{}
}

func (r *SeldonModelServe) Apply(ctx context.Context, md *modelsv1beta1.ModelDeployment) error {
	sd, err := r.convert(ctx, md)
	if err != nil {
		return err
	}
	if err := controllerutil.SetOwnerReference(md, sd, r.Client.Scheme()); err != nil {
		return err
	}
	coopy := sd.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, sd, func() error {
		sd.Annotations = Mergekvs(coopy.Annotations, sd.Annotations)
		sd.Labels = Mergekvs(coopy.Labels, sd.Labels)
		sd.Spec = coopy.Spec
		return nil
	})
	if err != nil {
		return err
	}

	if err := r.completeStatusURL(ctx, md, sd); err != nil {
		return err
	}

	md.Status.RawStatus = ToRawExtension(sd.Status)
	md.Status.Message = sd.Status.Description
	// fill phase
	switch sd.Status.State {
	case machinelearningv1.StatusStateAvailable:
		md.Status.Phase = modelsv1beta1.Running
	case machinelearningv1.StatusStateFailed:
		md.Status.Phase = modelsv1beta1.Failed
	default:
		md.Status.Phase = modelsv1beta1.Pending
	}
	return nil
}

func (r *SeldonModelServe) completeStatusURL(ctx context.Context, md *modelsv1beta1.ModelDeployment, sd *machinelearningv1.SeldonDeployment) error {
	// find same name ingress
	ingress := &networkingv1.Ingress{}
	// ignore error

	u := &url.URL{
		Scheme: r.IngressScheme,
		Host:   r.IngressHost,
		Path:   getIngressPath(ctx, r.Client, md),
	}

	_ = r.Client.Get(ctx, client.ObjectKey{Name: md.Name, Namespace: md.Namespace}, ingress)
	for _, rule := range ingress.Spec.Rules {
		if host := rule.Host; host != "" {
			u.Host = host
			break
		}
	}
	if gatewayName := md.Spec.Ingress.GatewayName; gatewayName != "" {
		gateway := &gemsv1beta1.TenantGateway{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: gatewayName}, gateway); err != nil {
			return err
		}
		for _, gatewayport := range gateway.Status.Ports {
			if gatewayport.Name == u.Scheme {
				u.Host += ":" + strconv.Itoa(int(gatewayport.NodePort))
				break
			}
		}
	}

	if md.Spec.Server.Protocol != "" {
		if address := sd.Status.Address; address != nil {
			if sdurl, err := url.Parse(address.URL); err == nil {
				u.Path += sdurl.Path
			}
		}
	}
	md.Status.URL = u.String()
	return nil
}

const ModelContainerName = "model"

// nolint: funlen
func (r *SeldonModelServe) convert(ctx context.Context, md *modelsv1beta1.ModelDeployment) (*machinelearningv1.SeldonDeployment, error) {
	sd := &machinelearningv1.SeldonDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        md.Name,
			Namespace:   md.Namespace,
			Annotations: md.Annotations,
			Labels:      md.Labels,
		},
		Spec: machinelearningv1.SeldonDeploymentSpec{
			Protocol:    machinelearningv1.Protocol(md.Spec.Server.Protocol),
			Annotations: md.Annotations,
			Predictors: []machinelearningv1.PredictorSpec{
				{
					Name:            "predictor",
					EngineResources: md.Spec.Server.Resources,
					Replicas:        md.Spec.Replicas,
					Annotations: map[string]string{
						// nolint: nosnakecase
						machinelearningv1.ANNOTATION_NO_ENGINE: isNoEngineKind(md.Spec.Server.Kind),
					},
					Graph: machinelearningv1.PredictiveUnit{
						Name:                    ModelContainerName,
						Implementation:          implOf(md.Spec.Server.Kind),
						Parameters:              paramsOf(md.Spec.Server.Parameters),
						ModelURI:                modelURIWithLicense(md.Spec.Model.URL, md.Spec.Model.License),
						StorageInitializerImage: md.Spec.Server.StorageInitializerImage,
					},
					ComponentSpecs: []*machinelearningv1.SeldonPodSpec{
						{
							Spec: completePod(md),
						},
					},
				},
			},
		},
	}

	ingressclass, err := r.getIngressClass(ctx, md)
	if err != nil {
		return nil, err
	}
	sd.Spec.Annotations["seldon.io/ingress-class-name"] = ingressclass
	sd.Spec.Annotations["seldon.io/ingress-host"] = md.Spec.Ingress.Host
	sd.Spec.Annotations["seldon.io/ingress-path"] = getIngressPath(ctx, r.Client, md) + "/(.*)"
	sd.Spec.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$1"
	return sd, nil
}

// nolint: funlen,gocognit
func completePod(md *modelsv1beta1.ModelDeployment) corev1.PodSpec {
	podspec := corev1.PodSpec{}
	if val := md.Spec.Server.PodSpec; val != nil {
		podspec = *val
	}
	CreateOrUpdateContainer(&podspec, ModelContainerName, func(c *v1.Container) {
		// add mounts
		for i, mount := range md.Spec.Server.Mounts {
			if mount.Kind == modelsv1beta1.SimpleVolumeMountKindModel {
				modelInitializerVolumeName := nameWithSuffix(c.Name, ModelInitializerVolumeSuffix)
				createOrUpdateMountPath(c, modelInitializerVolumeName, mount.MountPath)
				continue
			}
			otherMountName := "mount-" + strconv.Itoa(i)
			createOrUpdateMountPath(c, otherMountName, mount.MountPath)
			createOrUpdateVolume(&podspec, otherMountName, func() corev1.VolumeSource {
				switch mount.Kind {
				case modelsv1beta1.SimpleVolumeMountKindConfigMap:
					return corev1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: mount.Source}}}
				case modelsv1beta1.SimpleVolumeMountKindSecret:
					return corev1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: mount.Source}}
				case modelsv1beta1.SimpleVolumeMountKindEmptyDir:
					return corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
				case modelsv1beta1.SimpleVolumeMountKindHostPath:
					return corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: mount.Source}}
				case modelsv1beta1.SimpleVolumeMountKindPVC:
					return corev1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: mount.Source}}
				default:
					return corev1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: mount.Source}}
				}
			}())
		}

		if c.Resources.Limits == nil {
			c.Resources.Limits = md.Spec.Server.Resources.Limits
		}
		if c.Resources.Requests == nil {
			c.Resources.Requests = md.Spec.Server.Resources.Requests
		}
		if c.Args == nil {
			c.Args = md.Spec.Server.Args
		}
		if c.Image == "" {
			c.Image = md.Spec.Server.Image
		}
		if c.Args == nil {
			c.Args = md.Spec.Server.Args
		}
		if c.Ports == nil {
			c.Ports = md.Spec.Server.Ports
		}
		if c.Command == nil {
			c.Command = md.Spec.Server.Command
		}
		if c.Env == nil {
			c.Env = md.Spec.Server.Env
		}
		if md.Spec.Server.Privileged {
			if c.SecurityContext == nil {
				c.SecurityContext = &v1.SecurityContext{
					RunAsUser:  pointer.Int64(0),
					RunAsGroup: pointer.Int64(0),
				}
			}
			c.SecurityContext.Privileged = pointer.Bool(true)
		}
	})

	return podspec
}

func CreateOrUpdateContainer(pod *corev1.PodSpec, conname string, oncontainer func(c *corev1.Container)) {
	for i := range pod.Containers {
		if pod.Containers[i].Name == conname {
			oncontainer(&pod.Containers[i])
			return
		}
	}
	mainContainer := corev1.Container{Name: conname}
	oncontainer(&mainContainer)
	pod.Containers = append(pod.Containers, mainContainer)
}

func getIngressPath(ctx context.Context, cli client.Client, md *modelsv1beta1.ModelDeployment) string {
	ns := &corev1.Namespace{}
	_ = cli.Get(ctx, client.ObjectKey{Name: md.Namespace}, ns)
	if env := ns.Labels[gemlabels.LabelEnvironment]; env != "" {
		return "/" + env + "/" + md.Name
	}
	return "/" + md.Namespace + "/" + md.Name
}

func (r *SeldonModelServe) getIngressClass(ctx context.Context, md *modelsv1beta1.ModelDeployment) (string, error) {
	if md.Spec.Ingress.ClassName != "" {
		return md.Spec.Ingress.ClassName, nil
	}
	if getewayname := md.Spec.Ingress.GatewayName; getewayname != "" {
		gateway := &gemsv1beta1.TenantGateway{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: getewayname}, gateway); err != nil {
			return "", err
		}
		return gateway.Spec.IngressClass, nil
	}
	return "", nil
}

func implOf(name string) *machinelearningv1.PredictiveUnitImplementation {
	implName := strings.ToUpper(strings.Replace(name, "-", "_", -1))
	impl := machinelearningv1.PredictiveUnitImplementation(implName)
	return &impl
}

func paramsOf(params []modelsv1beta1.Parameter) []machinelearningv1.Parameter {
	p := make([]machinelearningv1.Parameter, 0, len(params))
	for _, item := range params {
		p = append(p, machinelearningv1.Parameter{Name: item.Name, Value: item.Value, Type: "STRING"})
	}
	return p
}

func isNoEngineKind(impl string) string {
	return strconv.FormatBool(impl == "")
}

func modelURIWithLicense(uri, license string) string {
	if license == "" {
		return uri
	}
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Sprintf("%s?license=%s", uri, license)
	}
	q := u.Query()
	q.Set("license", license)
	u.RawQuery = q.Encode()
	return u.String()
}

const ModelInitializerVolumeSuffix = "provision-location"

func createOrUpdateMountPath(c *corev1.Container, name, mountpath string) {
	if mountpath == "" || name == "" {
		return
	}
	for i, v := range c.VolumeMounts {
		if v.Name == name {
			c.VolumeMounts[i].MountPath = mountpath
			return
		}
	}
	c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{Name: name, MountPath: mountpath})
}

func createOrUpdateVolume(podspec *corev1.PodSpec, name string, vs corev1.VolumeSource) {
	if name == "" {
		return
	}
	for i, v := range podspec.Volumes {
		if v.Name == name {
			podspec.Volumes[i].VolumeSource = vs
			return
		}
	}
	podspec.Volumes = append(podspec.Volumes, v1.Volume{Name: name, VolumeSource: vs})
}

func nameWithSuffix(name string, suffix string) string {
	volumeName := name + "-" + suffix
	// kubernetes names limited to 63
	if len(volumeName) > 63 {
		volumeName = volumeName[0:63]
		volumeName = strings.TrimSuffix(volumeName, "-")
	}
	return volumeName
}
