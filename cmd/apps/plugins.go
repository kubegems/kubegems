package apps

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/service/handlers/application"
	"kubegems.io/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

type GlobalPluginsOptions struct {
	Directory string
}

func NewPluginCmd() *cobra.Command {
	globalOptions := &GlobalPluginsOptions{
		Directory: "deploy/plugins",
	}
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "plugins commands",
	}
	cmd.PersistentFlags().StringVarP(&globalOptions.Directory, "directory", "d", globalOptions.Directory, "cache directory")

	cmd.AddCommand(NewPluginTemplateCmd(globalOptions))
	cmd.AddCommand(NewPluginsDownloadCmd(globalOptions))
	return cmd
}

func NewPluginTemplateCmd(global *GlobalPluginsOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "template plugin",
		Example: `
		kubegem plugins template deploy/plugins/kubegem-local-stack
		`,
		RunE: func(cmd *cobra.Command, pathes []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			pm := controllers.NewPluginManager(nil, &controllers.PluginOptions{
				PluginsDir: global.Directory,
			})

			for _, path := range pathes {
				fi, err := os.Stat(path)
				if err != nil {
					return err
				}
				if !fi.IsDir() {
					return fmt.Errorf("%s is not a directory", path)
				}

				plugin := &pluginsv1beta1.Plugin{
					ObjectMeta: metav1.ObjectMeta{
						Name:      filepath.Base(path),
						Namespace: "default",
					},
					Spec: pluginsv1beta1.PluginSpec{
						Enabled: true,
						Kind:    controllers.DetectPluginType(path),
					},
				}

				resources, err := pm.Template(ctx, plugin)
				if err != nil {
					return err
				}
				for _, r := range resources {
					raw, err := yaml.Marshal(r)
					if err != nil {
						return err
					}
					fmt.Print(string(raw))
					fmt.Println("---")
				}
			}
			return nil
		},
	}
	return cmd
}

func NewPluginsDownloadCmd(global *GlobalPluginsOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download",
		Short:   "download plugins",
		Example: `kubegem plugins download deploy/plugins-core.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			ctx = logr.NewContext(ctx, zap.New(zap.UseDevMode(true)))

			for _, path := range args {
				var filecontent []byte
				var err error
				if path == "-" {
					filecontent, err = io.ReadAll(os.Stdin)
					if err != nil {
						return err
					}
				} else {
					filecontent, err = os.ReadFile(path)
					if err != nil {
						return err
					}
				}

				pm := controllers.NewPluginManager(nil, &controllers.PluginOptions{
					PluginsDir: global.Directory,
				})

				objs, err := kube.SplitYAMLTyped(filecontent)
				if err != nil {
					return err
				}
				for _, obj := range objs {
					apiplugin, ok := obj.(*pluginsv1beta1.Plugin)
					if !ok {
						continue
					}

					if err := pm.Download(ctx, apiplugin); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	return cmd
}

func parseImage(obj runtime.Object) []string {
	if uns, ok := obj.(*unstructured.Unstructured); ok {
		// covert to typed obj
		raw, err := yaml.Marshal(uns)
		if err != nil {
			return []string{}
		}
		typedobj, err := application.DecodeResource(raw)
		if err != nil {
			return []string{}
		}
		return application.ParseImagesFrom(typedobj)
	}
	if typedobj, ok := obj.(client.Object); ok {
		return application.ParseImagesFrom(typedobj)
	}
	return []string{}
}
