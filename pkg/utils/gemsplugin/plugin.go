package gemsplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	pluginscommon "kubegems.io/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PluginState struct {
	Annotations map[string]string      `json:"annotations"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description"`
	Healthy     bool                   `json:"healthy"`
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Version     string                 `json:"version"`
	Message     string                 `json:"message"`
	Values      map[string]interface{} `json:"values"`
}

type ListPluginOptions struct {
	WithHealthy bool
}

type ListPluginOption func(*ListPluginOptions)

func WithHealthy(b bool) ListPluginOption {
	return func(lpo *ListPluginOptions) {
		lpo.WithHealthy = b
	}
}

func ListPlugins(ctx context.Context, cli client.Client, options ...ListPluginOption) (map[string]interface{}, []PluginState, error) {
	opt := ListPluginOptions{
		WithHealthy: true,
	}
	for _, option := range options {
		option(&opt)
	}
	allinoneplugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginscommon.KubeGemsLocalPluginsName,
			Namespace: pluginscommon.KubeGemsLocalPluginsNamespace,
		},
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(allinoneplugin), allinoneplugin); err != nil {
		return nil, nil, err
	}
	var plugins map[string]PluginState
	if err := json.Unmarshal(allinoneplugin.Spec.Values.Raw, &plugins); err != nil {
		return nil, nil, err
	}
	result := []PluginState{}
	for name, plugin := range plugins {
		if name == "global" {
			continue
		}
		plugin.Name = name
		if opt.WithHealthy {
			checkHealthy(ctx, cli, &plugin)
		}
		result = append(result, plugin)
	}
	globalVals, _ := allinoneplugin.Spec.Values.Object["global"].(map[string]interface{})
	return globalVals, result, nil
}

func checkHealthy(ctx context.Context, cli client.Client, plugin *PluginState) {
	if !plugin.Enabled {
		plugin.Healthy = false
		return // plugin is not enabled
	}
	msgs := []string{}
	if annotations := plugin.Annotations; annotations != nil {
		for _, checkExpression := range strings.Split(plugin.Annotations[pluginscommon.AnnotationHealthCheck], ",") {
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

func EnablePlugin(ctx context.Context, cli client.Client, name string, enable bool) error {
	allinoneplugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginscommon.KubeGemsLocalPluginsName,
			Namespace: pluginscommon.KubeGemsLocalPluginsNamespace,
		},
	}
	patchData := fmt.Sprintf(`{"spec":{"values":{"%s":{"enabled":%t}}}}`, name, enable)
	patch := client.RawPatch(types.MergePatchType, []byte(patchData))
	return cli.Patch(ctx, allinoneplugin, patch)
}
