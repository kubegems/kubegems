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

package statistics

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"kubegems.io/kubegems/pkg/utils/resourcequota"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterWorkloadStatistics map[string]int

func GetWorkloadsStatistics(ctx context.Context, cli client.Client) ClusterWorkloadStatistics {
	ret := map[string]int{}

	deployments := &appsv1.DeploymentList{}
	_ = cli.List(ctx, deployments)
	ret[resourcequota.ResourceDeployments.String()] = len(deployments.Items)

	statefulsetCounter := &appsv1.StatefulSetList{}
	_ = cli.List(ctx, statefulsetCounter)
	ret[resourcequota.ResourceStatefulSets.String()] = len(statefulsetCounter.Items)

	daemonsetCounter := &appsv1.DaemonSetList{}
	_ = cli.List(ctx, daemonsetCounter)
	ret[resourcequota.ResourceDaemonsets.String()] = len(daemonsetCounter.Items)

	podCounter := &corev1.PodList{}
	_ = cli.List(ctx, podCounter)
	ret[corev1.ResourcePods.String()] = len(podCounter.Items)

	configmapCounter := &corev1.ConfigMapList{}
	_ = cli.List(ctx, configmapCounter)
	ret[resourcequota.ResourceConfigMaps.String()] = len(configmapCounter.Items)

	secretCounter := &corev1.SecretList{}
	_ = cli.List(ctx, secretCounter)
	ret[resourcequota.ResourceSecrets.String()] = len(secretCounter.Items)

	pvcCounter := &corev1.PersistentVolumeList{}
	_ = cli.List(ctx, pvcCounter)
	ret[resourcequota.ResourcePersistentVolumeClaims.String()] = len(pvcCounter.Items)

	serviceCounter := &corev1.ServiceList{}
	_ = cli.List(ctx, serviceCounter)
	ret[resourcequota.ResourceServices.String()] = len(serviceCounter.Items)

	cronjobCounter := &batchv1beta1.CronJobList{}
	_ = cli.List(ctx, cronjobCounter)
	ret[resourcequota.ResourceCronJobs.String()] = len(cronjobCounter.Items)

	jobCounter := &batchv1.JobList{}
	_ = cli.List(ctx, jobCounter)
	ret[resourcequota.ResourceJobs.String()] = len(jobCounter.Items)

	namespaceCounter := &corev1.NamespaceList{}
	_ = cli.List(ctx, namespaceCounter)
	ret["namespace"] = len(namespaceCounter.Items)

	nodeCounter := &corev1.NodeList{}
	_ = cli.List(ctx, nodeCounter)
	ret["node"] = len(nodeCounter.Items)

	return ret
}
