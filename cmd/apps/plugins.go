package apps

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

func NewPluginCmd() *cobra.Command {
	globalOptions := controllers.NewDefaultOPluginptions()
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "plugins commands",
	}
	cmd.PersistentFlags().StringVarP(&globalOptions.CacheDir, "cache", "c", globalOptions.CacheDir, "cache download plugins to this directory")
	cmd.PersistentFlags().StringSliceVarP(&globalOptions.SearchDirs, "directory", "d", globalOptions.SearchDirs, "search plugins in directories")

	cmd.AddCommand(NewPluginTemplateCmd(globalOptions))
	cmd.AddCommand(NewPluginsDownloadCmd(globalOptions))
	return cmd
}

// nolint: gocognit
func NewPluginTemplateCmd(global *controllers.PluginOptions) *cobra.Command {
	pretty := false
	cmd := &cobra.Command{
		Use:   "template",
		Short: "template plugin",
		Example: `
		kubegem plugins template deploy/plugins/kubegem-local-stack
		kubegem plugins template deploy/plugins-core.yaml
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			pm := controllers.NewPluginManager(nil, global)
			// to avoid log message mix in yaml contents
			log.Default().SetOutput(io.Discard)
			for _, path := range args {
				var content []byte
				var err error
				// nolint:nestif
				if path == "-" {
					content, err = io.ReadAll(os.Stdin)
					if err != nil {
						return err
					}
				} else {
					fi, err := os.Stat(path)
					if err != nil {
						return err
					}
					if fi.IsDir() {
						// template current directory
						pm.Options.SearchDirs = append(pm.Options.SearchDirs, filepath.Dir(path))
						err := templatePrint(ctx, pm, &pluginsv1beta1.Plugin{
							ObjectMeta: metav1.ObjectMeta{Name: filepath.Base(path), Namespace: "default"},
							Spec:       pluginsv1beta1.PluginSpec{Enabled: true, Kind: controllers.DetectPluginType(path)},
						}, pretty)
						if err != nil {
							return err
						}
						continue
					}
					content, err = os.ReadFile(path)
					if err != nil {
						return err
					}
				}
				// template every plugin
				objs, err := kube.SplitYAMLFilterByExample[*pluginsv1beta1.Plugin](content)
				if err != nil {
					return err
				}
				for _, obj := range objs {
					if err := templatePrint(ctx, pm, obj, pretty); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&pretty, "pretty", "p", false, "pretty print")
	return cmd
}

func templatePrint(ctx context.Context, pm *controllers.PluginManager, plugin *pluginsv1beta1.Plugin, pretty bool) error {
	manifestdoc, err := pm.Template(ctx, plugin)
	if err != nil {
		return err
	}
	if !pretty {
		fmt.Print(string(manifestdoc))
		return nil
	}
	objects, err := kube.SplitYAMLFilterByExample[runtime.Object](manifestdoc)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		raw, err := yaml.Marshal(obj)
		if err != nil {
			return err
		}
		fmt.Println("---")
		fmt.Print(string(raw))
	}
	return nil
}

func NewPluginsDownloadCmd(global *controllers.PluginOptions) *cobra.Command {
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
				pm := controllers.NewPluginManager(nil, global)
				objs, err := kube.SplitYAMLFilterByExample[*pluginsv1beta1.Plugin](filecontent)
				if err != nil {
					return err
				}
				for _, obj := range objs {
					if err := pm.Download(ctx, obj); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	return cmd
}
