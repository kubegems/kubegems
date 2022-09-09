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

package helm

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

type ReleaseManager struct {
	Config *rest.Config
}

func NewHelmConfig(ctx context.Context, namespace string, cfg *rest.Config) (*action.Configuration, error) {
	baselog := logr.FromContextOrDiscard(ctx)
	logfunc := func(format string, v ...interface{}) {
		baselog.Info(fmt.Sprintf(format, v...))
	}

	cligetter := genericclioptions.NewConfigFlags(true)
	cligetter.WrapConfigFn = func(*rest.Config) *rest.Config {
		return cfg
	}

	config := &action.Configuration{}
	config.Init(cligetter, namespace, "", logfunc) // release storage namespace
	if kc, ok := config.KubeClient.(*kube.Client); ok {
		kc.Namespace = namespace // install to namespace
	}
	return config, nil
}

func TemplateChart(ctx context.Context, rlsname, namespace string, chartPath string, values map[string]any) ([]byte, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}
	install := action.NewInstall(&action.Configuration{})
	install.ReleaseName, install.Namespace = rlsname, namespace
	install.DryRun, install.DisableHooks, install.ClientOnly = true, true, true
	rls, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, err
	}
	return []byte(rls.Manifest), nil
}

func ApplyChart(ctx context.Context, cfg *rest.Config, rlsname, namespace string, chartPath string, values map[string]interface{}) (*release.Release, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("name", rlsname, "namespace", namespace)
	log.Info("loading chart")
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}
	if rlsname == "" {
		rlsname = chart.Name()
	}
	helmcfg, err := NewHelmConfig(ctx, namespace, cfg)
	if err != nil {
		return nil, err
	}
	existRelease, err := action.NewGet(helmcfg).Run(rlsname)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, err
		}
		// not install, install it now
		log.Info("installing", "values", values)
		install := action.NewInstall(helmcfg)
		install.ReleaseName, install.Namespace = rlsname, namespace
		install.CreateNamespace = true
		return install.RunWithContext(ctx, chart, values)
	}
	// check should upgrade
	if existRelease.Info.Status == release.StatusDeployed && equalMapValues(existRelease.Config, values) {
		log.Info("already uptodate", "values", values)
		return existRelease, nil
	}
	log.Info("upgrading", "old", existRelease.Config, "new", values)
	client := action.NewUpgrade(helmcfg)
	client.Namespace = namespace
	client.ResetValues = true
	// client.MaxHistory = 5  // there is a bug,do not use it.

	removeHistories(ctx, helmcfg.Releases, rlsname, 2)

	return client.RunWithContext(ctx, rlsname, chart, values)
}

func removeHistories(ctx context.Context, storage *storage.Storage, name string, max int) error {
	rlss, err := storage.History(name)
	if err != nil {
		return err
	}
	if max <= 0 {
		max = 1
	}

	// newest to old
	slices.SortFunc(rlss, func(a, b *release.Release) bool {
		return a.Version > b.Version
	})

	var lastDeployed *release.Release
	toDelete := []*release.Release{}
	for _, rls := range rlss {
		if rls.Info.Status == release.StatusDeployed && lastDeployed == nil {
			lastDeployed = rls
			continue
		}
		// once we have enough releases to delete to reach the max, stop
		// all - deleted = max
		if len(rlss)-len(toDelete) == max {
			break
		}
		toDelete = append(toDelete, rls)
	}
	for _, todel := range toDelete {
		if _, err := storage.Delete(todel.Name, todel.Version); err != nil {
			return err
		}
	}
	return nil
}

func equalMapValues(a, b map[string]interface{}) bool {
	return (len(a) == 0 && len(b) == 0) || reflect.DeepEqual(a, b)
}

func RemoveChart(ctx context.Context, cfg *rest.Config, rlsname, namespace string) (*release.Release, error) {
	log := logr.FromContextOrDiscard(ctx)
	helmcfg, err := NewHelmConfig(ctx, namespace, cfg)
	if err != nil {
		return nil, err
	}
	exist, err := action.NewGet(helmcfg).Run(rlsname)
	if err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, err
		}
		return nil, nil
	}
	log.Info("uninstalling")
	uninstall := action.NewUninstall(helmcfg)
	uninstalledRelease, err := uninstall.Run(exist.Name)
	if err != nil {
		return nil, err
	}
	return uninstalledRelease.Release, nil
}
