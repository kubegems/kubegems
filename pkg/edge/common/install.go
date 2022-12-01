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

package common

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultEdgeAgentImage = "docker.io/kubegems/kubegems-edge-agent:latest"

// nolint: gomnd,funlen
func RenderManifets(uid string, image string, edgehubaddress string, certs v1beta1.Certs) []client.Object {
	if image == "" {
		image = DefaultEdgeAgentImage
	}
	installname, installnamespace := "kubegems-edge-agent", "kubegems-edge"
	commonlabels := map[string]string{
		"app.kubernetes.io/instance":  "kubegems-edge",
		"app.kubernetes.io/name":      "kubegems-edge",
		"app.kubernetes.io/component": "agent",
	}
	certsSecretname := installname + "-secret"
	return []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: installnamespace,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      installname,
				Namespace: installnamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: commonlabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: commonlabels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: installname,
						Containers: []corev1.Container{
							{
								Name:  "agent",
								Image: image,
								Args: []string{
									"--listen=:8080",
									"--edgehubaddr=" + edgehubaddress,
									"--clientid=" + uid,
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8080,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "certs",
										MountPath: "/app/certs",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "certs",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: certsSecretname,
									},
								},
							},
						},
					},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      installname,
				Namespace: installnamespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: commonlabels,
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: intstr.FromString("http"),
					},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certsSecretname,
				Namespace: installnamespace,
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				corev1.TLSCertKey:              certs.Cert,
				corev1.TLSPrivateKeyKey:        certs.Key,
				corev1.ServiceAccountRootCAKey: certs.CA,
			},
		},
		// cluster RBAC
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      installname,
				Namespace: installnamespace,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: installname,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      installname,
					Namespace: installnamespace,
				},
			},
		},
	}
}
