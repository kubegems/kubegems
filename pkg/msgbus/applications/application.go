package applications

import (
	"context"
	"fmt"
	"math"
	"time"

	argoapplication "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/msgbus/switcher"
	"kubegems.io/kubegems/pkg/service/handlers/application"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/retry"
)

var DefaultBackoff = wait.Backoff{
	Steps:    math.MaxInt32,
	Duration: 5 * time.Second,
	Factor:   2,
	Jitter:   0.1,
	Cap:      30 * time.Second,
}

type ApplicationMessageCollector struct {
	argo      *argo.Client
	ms        *switcher.MessageSwitcher
	messageCh chan *msgbus.NotifyMessage
}

func RunApplicationCollector(ctx context.Context, ms *switcher.MessageSwitcher, argo *argo.Client) error {
	c := &ApplicationMessageCollector{
		argo:      argo,
		ms:        ms,
		messageCh: make(chan *msgbus.NotifyMessage, 1000),
	}
	return c.Run(ctx)
}

func (c *ApplicationMessageCollector) Run(ctx context.Context) error {
	// producer
	// 重试属于正常控制流程，不算错误
	log.FromContextOrDiscard(ctx).Info("start watching argo application...")

	go retry.Always(func() error {
		return c.runOriginWay(ctx)
	})

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-c.messageCh:
			c.ms.DispatchMessage(msg)
		}
	}
}

func (c *ApplicationMessageCollector) runOriginWay(ctx context.Context) error {
	closer, cli, err := c.argo.ArgoCDcli.NewApplicationClient()
	if err != nil {
		return err
	}
	defer closer.Close()

	// 在argo cd webui中watch也为30s
	// 如果 name 和 reversion 均未设置，则会触发一次 list 时间类型为 Add
	watcher, err := cli.Watch(ctx, &argoapplication.ApplicationQuery{ResourceVersion: "0"})
	if err != nil {
		return err
	}

	for {
		event, err := watcher.Recv()
		if err != nil {
			return err
		}
		switch event.Type {
		case watch.Error:
			return fmt.Errorf("watch error: %v", event)
		default:
			app := &event.Application
			msg := &msgbus.NotifyMessage{
				MessageType: msgbus.Changed,
				InvolvedObject: &msgbus.InvolvedObject{
					Cluster: application.DecodeArgoClusterName(
						app.Spec.Destination.Name,
					),
					Group:   v1alpha1.ApplicationSchemaGroupVersionKind.Group,
					Kind:    v1alpha1.ApplicationSchemaGroupVersionKind.Kind,
					Version: v1alpha1.ApplicationSchemaGroupVersionKind.Version,
					NamespacedName: msgbus.NamespacedNameFrom(
						app.Spec.Destination.Namespace,
						app.Labels[gemlabels.LabelApplication],
					),
				},
				EventKind: func() msgbus.EventKind {
					switch event.Type {
					case watch.Added:
						return msgbus.Add
					case watch.Modified:
						return msgbus.Update
					case watch.Deleted:
						return msgbus.Delete
					default:
						return msgbus.Update
					}
				}(),
				Content: application.CompleteDeploiedManifestRuntime(app, &application.DeploiedManifest{}),
			}
			c.messageCh <- msg
		}
	}
}
