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

package apps

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pluginv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	"kubegems.io/kubegems/pkg/installer/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPluginCmd() *cobra.Command {
	cmd := NewBundleControllerCmd()
	return cmd
}

func NewBundleControllerCmd() *cobra.Command {
	globalOptions := bundle.NewDefaultOptions()
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "commands of plugins",
	}
	cmd.AddCommand(
		NewDownloadCmd(globalOptions),
		NewTemplateCmd(globalOptions),
	)
	cmd.PersistentFlags().StringVarP(&globalOptions.CacheDir, "cache-dir", "c", globalOptions.CacheDir, "cache directory")
	cmd.PersistentFlags().StringSliceVarP(&globalOptions.SearchDirs, "search-dir", "s", globalOptions.SearchDirs, "search bundles in directory")
	return cmd
}

func NewTemplateCmd(options *bundle.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "template a bundle",
		Example: `
# template a helm bundle into stdout
bundle -c bundles template helm-bundle.yaml
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			apply := bundle.NewDefaultApply(nil, nil, options)
			return forBundleInPathes(args, func(bundle *pluginv1beta1.Plugin) error {
				content, err := apply.Template(ctx, bundle)
				if err != nil {
					return err
				}
				fmt.Print(string(content))
				return nil
			})
		},
	}
	return cmd
}

func NewDownloadCmd(options *bundle.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "download a bundle",
		Example: `
# download a helm bundle into bundles
bundle -c bundles download helm-bundle.yaml
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			zapl, _ := zap.NewDevelopment()
			ctx = logr.NewContext(ctx, zapr.NewLogger(zapl))

			apply := bundle.NewDefaultApply(nil, nil, options)

			return forBundleInPathes(args, func(bundle *pluginv1beta1.Plugin) error {
				_, err := apply.Download(ctx, bundle)
				return err
			})
		},
	}
	return cmd
}

func forBundleInPathes(pathes []string, fun func(*pluginv1beta1.Plugin) error) error {
	return ForBundleInPathes(pathes, PluginFromDir, func(plugin *pluginv1beta1.Plugin) error {
		return fun(plugin)
	})
}

func ForBundleInPathes[T client.Object](pathes []string, readdir func(string) T, fun func(T) error) error {
	if len(pathes) == 1 && pathes[0] == "-" {
		objs, err := utils.SplitYAMLFilterd[T](os.Stdin)
		if err != nil {
			return err
		}
		for _, obj := range objs {
			if err := fun(obj); err != nil {
				return err
			}
		}
		return nil
	}

	for _, path := range pathes {
		fi, err := os.Lstat(path)
		if err != nil {
			return err
		}

		var objs []T
		if fi.IsDir() {
			objs = []T{readdir(path)}
		} else {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			objs, err = utils.SplitYAMLFilterd[T](bytes.NewReader(content))
			if err != nil {
				return err
			}
		}
		for _, obj := range objs {
			if err := fun(obj); err != nil {
				return err
			}
		}
	}
	return nil
}

func PluginFromDir(dir string) *pluginv1beta1.Plugin {
	return (*pluginv1beta1.Plugin)(unsafe.Pointer(BundleFromDir(dir)))
}

func BundleFromDir(dir string) *pluginv1beta1.Plugin {
	// detect kind
	kind := pluginv1beta1.BundleKindTemplate
	if _, err := os.Stat(filepath.Join(dir, "Chart.yaml")); err == nil {
		kind = pluginv1beta1.BundleKindHelm
	} else if _, err := os.Stat(filepath.Join(dir, "kustomization.yaml")); err == nil {
		kind = pluginv1beta1.BundleKindKustomize
	}
	dir, _ = filepath.Abs(dir)
	return &pluginv1beta1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      filepath.Base(dir),
			Namespace: "default",
		},
		Spec: pluginv1beta1.PluginSpec{
			Kind: kind,
			URL:  "file://" + dir,
		},
	}
}
