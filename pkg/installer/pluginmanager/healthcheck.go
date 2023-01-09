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

package pluginmanager

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckHealthy(ctx context.Context, cli client.Client, plugin *PluginVersion) {
	if !plugin.Enabled {
		plugin.Healthy = false
		return // plugin is not enabled
	}
	msgs := []string{}
	for _, checkExpression := range strings.Split(plugin.HelathCheck, ",") {
		splits := strings.Split(checkExpression, "/")
		const lenResourceAndName = 2
		if len(splits) != lenResourceAndName {
			continue
		}
		resource, nameregexp := splits[0], splits[1]
		if err := checkHealthItem(ctx, cli, resource, plugin.Namespace, nameregexp); err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	if len(msgs) > 0 {
		plugin.Message = strings.Join(msgs, ",")
		plugin.Healthy = false
	} else {
		plugin.Healthy = true
	}
}

func checkHealthItem(ctx context.Context, cli client.Client, resource, namespace, nameregexp string) error {
	switch {
	case strings.Contains(strings.ToLower(resource), "deployment"):
		deploymentList := &appsv1.DeploymentList{}
		_ = cli.List(ctx, deploymentList, client.InNamespace(namespace))
		return matchAndCheck(deploymentList.Items, nameregexp, func(dep appsv1.Deployment) error {
			if dep.Status.ReadyReplicas != dep.Status.Replicas {
				return fmt.Errorf("Deployment %s is not ready", dep.Name)
			}
			return nil
		})
	case strings.Contains(resource, "statefulset"):
		statefulsetList := &appsv1.StatefulSetList{}
		_ = cli.List(ctx, statefulsetList, client.InNamespace(namespace))
		return matchAndCheck(statefulsetList.Items, nameregexp, func(sts appsv1.StatefulSet) error {
			if sts.Status.ReadyReplicas != sts.Status.Replicas {
				return fmt.Errorf("StatefulSet %s is not ready", sts.Name)
			}
			return nil
		})
	case strings.Contains(resource, "daemonset"):
		daemonsetList := &appsv1.DaemonSetList{}
		_ = cli.List(ctx, daemonsetList, client.InNamespace(namespace))
		return matchAndCheck(daemonsetList.Items, nameregexp, func(ds appsv1.DaemonSet) error {
			if ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
				return fmt.Errorf("DaemonSet %s is not ready", ds.Name)
			}
			return nil
		})
	}
	return nil
}

func matchAndCheck[T any](list []T, exp string, check func(T) error) error {
	var msgs []string
	for _, item := range list {
		obj, ok := any(item).(client.Object)
		if !ok {
			obj, ok = any(&item).(client.Object)
		}
		if !ok {
			continue
		}
		match, _ := regexp.MatchString(exp, obj.GetName())
		if !match {
			continue
		}
		if err := check(item); err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	if len(msgs) > 0 {
		return fmt.Errorf("%s", strings.Join(msgs, ","))
	}
	return nil
}
