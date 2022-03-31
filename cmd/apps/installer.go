package apps

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
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
	cmd.AddCommand(NewTemplateCmd())
	return cmd
}

func NewTemplateCmd() *cobra.Command {
	namespace := "default"
	cmd := &cobra.Command{
		Use:   "template",
		Short: "run template",
		Example: `
		kubegem installer template --namespace=my-namespace sample-plugin
		`,
		RunE: func(cmd *cobra.Command, pathes []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			path := ""
			if len(pathes) == 0 {
				path = ""
			} else {
				path = pathes[0]
			}

			plugin := controllers.Plugin{
				Namespace: namespace,
				Path:      path,
				Name:      filepath.Base(path),
			}
			resources, err := controllers.TemplatesBuildPlugin(ctx, plugin)
			if err != nil {
				return err
			}
			for _, resource := range resources {
				content, err := yaml.Marshal(resource)
				if err != nil {
					fmt.Printf("%v\n", err.Error())
				}
				fmt.Printf("%s\n", string(content))
				fmt.Println("---")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&namespace, "namespace", "n", namespace, "template namespace")
	return cmd
}
