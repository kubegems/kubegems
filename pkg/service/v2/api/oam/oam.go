package oam

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	oamv1beta1 "github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (o *OAM) ListApplications(req *restful.Request, resp *restful.Response) {
	o.AppRefFunc(req, resp, func(ctx context.Context, cli client.Client, ref AppRef) (interface{}, error) {
		list := &oamv1beta1.ApplicationList{}
		if err := cli.List(ctx, list, client.InNamespace(ref.Namespace)); err != nil {
			return nil, err
		}
		return list, nil
	})
}
