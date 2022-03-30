package apps

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	pluginv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/installer"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/utils/config"
	"sigs.k8s.io/yaml"
)

func NewInstallerCmd() *cobra.Command {
	options := installer.NewDefaultOptions()
	cmd := &cobra.Command{
		Use:   "installer",
		Short: "run installer",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return installer.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	cmd.AddCommand(NewGoTemplateCmd())
	return cmd
}

func NewGoTemplateCmd() *cobra.Command {
	pluginpath := ""
	plugindefinition := ""
	cmd := &cobra.Command{
		Use:   "template",
		Short: "template plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			content, err := ioutil.ReadFile(plugindefinition)
			if err != nil {
				return err
			}
			plugin := &pluginv1beta1.Plugin{}
			if err := yaml.Unmarshal(content, plugin); err != nil {
				return err
			}

			ns := plugin.Spec.InstallNamespace
			if ns == "" {
				ns = plugin.Namespace
			}
			release := controllers.Release{Name: plugin.Name, Namespace: ns}
			resources, err := controllers.TemplatesBuild(ctx, pluginpath, release, controllers.UnmarshalValues(plugin.Spec.Values))
			if err != nil {
				return err
			}
			for _, resource := range resources {
				content, err := yaml.Marshal(resource)
				if err != nil {
					return err
				}
				fmt.Println(string(content))
				fmt.Println("---")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&pluginpath, "path", "p", "kubegems-local-stack", "plugin path")
	cmd.Flags().StringVarP(&plugindefinition, "definition", "d", "plugins-local-stack.yaml", "plugin definition yaml file")
	return cmd
}
