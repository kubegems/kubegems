package gemsplugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	bundlev1 "kubegems.io/bundle-controller/pkg/apis/bundle/v1beta1"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
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
	AppVersion  string                 `json:"appVersion"`
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

func ListPlugins(ctx context.Context, cli client.Client, options ...ListPluginOption) (map[string]string, []PluginState, error) {
	opt := ListPluginOptions{
		WithHealthy: true,
	}
	for _, option := range options {
		option(&opt)
	}
	plugins := &pluginsv1beta1.PluginList{}
	if err := cli.List(ctx, plugins, client.InNamespace(pluginscommon.KubeGemsLocalPluginsNamespace)); err != nil {
		return nil, nil, err
	}

	globalvalues := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginscommon.KubeGemsGlobalValuesConfigMapName,
			Namespace: pluginscommon.KubeGemsLocalPluginsNamespace,
		},
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(globalvalues), globalvalues); err != nil {
		// return nil, nil, err
	}
	result := []PluginState{}
	for _, bundle := range plugins.Items {
		if annotations := bundle.Annotations; annotations != nil {
			if _, ok := annotations[pluginscommon.AnnotationIgnoreOnDisabled]; ok {
				continue
			}
		}
		state := PluginState{
			Annotations: bundle.Annotations,
			Enabled:     !bundle.Spec.Disabled,
			Description: "",
			Healthy:     true,
			Name:        bundle.Name,
			Namespace:   bundle.Spec.InstallNamespace,
			Version:     bundle.Spec.Version,
			Values:      make(map[string]interface{}),
			AppVersion:  bundle.Annotations[pluginscommon.AnnotationAppVersion],
		}
		if state.Version == "" {
			state.Version = state.AppVersion
		}
		// health check
		if bundle.Status.Phase != bundlev1.PhaseInstalled {
			state.Healthy = false
			state.Message = bundle.Status.Message
		} else if opt.WithHealthy {
			checkHealthy(ctx, cli, &state)
		} else {
			state.Healthy = true
		}
		result = append(result, state)
	}
	return globalvalues.Data, result, nil
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
	plugin := &pluginsv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: pluginscommon.KubeGemsLocalPluginsNamespace},
	}
	patchData := fmt.Sprintf(`{"spec":{"disabled":%t}}`, !enable)
	patch := client.RawPatch(types.MergePatchType, []byte(patchData))
	return cli.Patch(ctx, plugin, patch)
}
