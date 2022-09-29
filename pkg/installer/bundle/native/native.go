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

package native

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type TemplateFun func(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) ([]byte, error)

type Apply struct {
	TemplateFun TemplateFun
	Cli         *utils.Apply
}

func New(cli client.Client, fun TemplateFun) *Apply {
	return &Apply{
		TemplateFun: fun,
		Cli:         &utils.Apply{Client: cli},
	}
}

func (p *Apply) Template(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) ([]byte, error) {
	return p.TemplateFun(ctx, bundle, into)
}

func (p *Apply) Apply(ctx context.Context, bundle *pluginsv1beta1.Plugin, into string) error {
	log := logr.FromContextOrDiscard(ctx)

	rendered, err := p.Template(ctx, bundle, into)
	if err != nil {
		return err
	}
	resources, err := utils.SplitYAML(rendered)
	if err != nil {
		return err
	}

	ns := bundle.Spec.InstallNamespace
	if ns == "" {
		ns = bundle.Namespace
	}
	// override namespace
	p.CorrectNamespaces(ctx, ns, p.Cli.Client, resources)

	diffresult := utils.Diff(bundle.Status.Resources, resources)
	if bundle.Status.Phase == pluginsv1beta1.PhaseInstalled &&
		utils.EqualMapValues(bundle.Status.Values.Object, bundle.Spec.Values.Object) &&
		len(diffresult.Creats) == 0 &&
		len(diffresult.Removes) == 0 {
		log.Info("all resources are already applied")
		return nil
	}
	managedResources, err := p.Cli.SyncDiff(ctx, diffresult, utils.NewDefaultSyncOptions())
	if err != nil {
		return err
	}
	bundle.Status.Resources = managedResources
	bundle.Status.Values = pluginsv1beta1.Values{Object: bundle.Spec.Values.Object}.FullFill()
	bundle.Status.Phase = pluginsv1beta1.PhaseInstalled
	bundle.Status.Version = bundle.Spec.Version
	bundle.Status.Namespace = ns
	now := metav1.Now()
	bundle.Status.UpgradeTimestamp = now
	if bundle.Status.CreationTimestamp.IsZero() {
		bundle.Status.CreationTimestamp = now
	}
	bundle.Status.Message = ""
	return nil
}

func (p *Apply) Remove(ctx context.Context, bundle *pluginsv1beta1.Plugin) error {
	managedResources, err := p.Cli.Sync(ctx, bundle.Status.Resources, nil, utils.NewDefaultSyncOptions())
	if err != nil {
		return err
	}
	bundle.Status.Resources = managedResources
	bundle.Status.Phase = pluginsv1beta1.PhaseDisabled
	bundle.Status.Message = ""
	return nil
}

func (p *Apply) CorrectNamespaces(ctx context.Context, ns string, cli client.Client, list []*unstructured.Unstructured) {
	log := logr.FromContextOrDiscard(ctx)
	for _, item := range list {
		scopeName, err := p.IsNamespacedScope(cli, item)
		if err != nil {
			log.Error(err, "get scope name", "kind", item.GetKind())
			continue
		}
		switch {
		case scopeName == apimeta.RESTScopeNameNamespace && item.GetNamespace() == "":
			item.SetNamespace(ns)
		case scopeName == apimeta.RESTScopeNameRoot && item.GetNamespace() != "":
			item.SetNamespace("")
		}
	}
}

func (p *Apply) IsNamespacedScope(cli client.Client, obj client.Object) (apimeta.RESTScopeName, error) {
	restmapper := cli.RESTMapper()
	scheme := cli.Scheme()
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return "", err
	}
	restmapping, err := restmapper.RESTMapping(gvk.GroupKind())
	if err != nil {
		return "", fmt.Errorf("failed to get restmapping: %w", err)
	}
	return restmapping.Scope.Name(), nil
}
