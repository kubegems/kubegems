package apps

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"kubegems.io/pkg/installer/controllers"
	"kubegems.io/pkg/service/handlers/application"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func NewPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "plugin commands",
	}
	cmd.AddCommand(NewPluginTemplateCmd())
	return cmd
}

func NewPluginTemplateCmd() *cobra.Command {
	const (
		outputImage = "image"
		outputYaml  = "yaml"
	)

	cachedir := "deploy/plugins"
	recursive := false
	output := outputYaml

	cmd := &cobra.Command{
		Use:   "template",
		Short: "template plugin",
		Example: `
		kubegem plugin template deploy/plugins/kubegem-local-stack
		`,
		RunE: func(cmd *cobra.Command, pathes []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			resources := []runtime.Object{}
			printfunc := func(obj runtime.Object) {
				resources = append(resources, obj)
			}
			for _, path := range pathes {
				if err := controllers.TemplatePlugins(ctx, path, cachedir, printfunc, recursive); err != nil {
					return err
				}
			}

			switch output {
			case outputImage:
				for _, obj := range resources {
					images := parseImage(obj)
					for _, image := range images {
						fmt.Println(image)
					}
				}
			case outputYaml:
				for _, obj := range resources {
					raw, err := yaml.Marshal(obj)
					if err != nil {
						continue
					}
					fmt.Print(string(raw))
					fmt.Println("---")
				}
			default:
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&cachedir, "directory", "d", cachedir, "cache directory")
	cmd.Flags().StringVarP(&output, "output", "o", output, "output format,supported formats: image, yaml")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", recursive, "template plugins in template result recursively")
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
