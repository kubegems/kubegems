package common

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ListOptions struct {
	Page     int
	Size     int
	Selector labels.Selector
}

type EdgeClusterStore interface {
	List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeCluster, error)
	Get(ctx context.Context, name string) (*v1beta1.EdgeCluster, error)
	Create(ctx context.Context, cluster *v1beta1.EdgeCluster) error
	Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeCluster) error) (*v1beta1.EdgeCluster, error)
	Delete(ctx context.Context, name string) (*v1beta1.EdgeCluster, error)
}

func NewLocalK8sStore(ns string) (*EdgeClusterK8sStore, error) {
	if ns == "" {
		ns = "kubegems-edge"
	}
	cli, err := kube.NewLocalClient()
	if err != nil {
		return nil, err
	}
	return &EdgeClusterK8sStore{C: cli, NS: ns}, nil
}

type EdgeClusterK8sStore struct {
	C  client.Client
	NS string
}

func (s *EdgeClusterK8sStore) List(ctx context.Context, options ListOptions) (int, []v1beta1.EdgeCluster, error) {
	list := &v1beta1.EdgeClusterList{}
	if err := s.C.List(ctx, list,
		client.InNamespace(s.NS),
		client.MatchingLabelsSelector{Selector: options.Selector},
	); err != nil {
		return 0, nil, err
	}
	if options.Page == 0 && options.Size == 0 {
		return len(list.Items), list.Items, nil
	} else {
		paged := response.NewTypedPage(list.Items, options.Page, options.Size, nil, nil)
		return len(list.Items), paged.List, nil
	}
}

func (s *EdgeClusterK8sStore) Get(ctx context.Context, name string) (*v1beta1.EdgeCluster, error) {
	ret := &v1beta1.EdgeCluster{}
	if err := s.C.Get(ctx, client.ObjectKey{Name: name, Namespace: s.NS}, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *EdgeClusterK8sStore) Create(ctx context.Context, edge *v1beta1.EdgeCluster) error {
	edge.SetNamespace(s.NS)
	return s.C.Create(ctx, edge)
}

func (s *EdgeClusterK8sStore) Update(ctx context.Context, name string, fun func(cluster *v1beta1.EdgeCluster) error) (*v1beta1.EdgeCluster, error) {
	obj := &v1beta1.EdgeCluster{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: s.NS,
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx, s.C, obj, func() error {
		return fun(obj)
	})
	return obj, err
}

func (s *EdgeClusterK8sStore) Delete(ctx context.Context, name string) (*v1beta1.EdgeCluster, error) {
	remove := &v1beta1.EdgeCluster{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: s.NS},
	}
	if err := s.C.Delete(ctx, remove); err != nil {
		return nil, err
	}
	return remove, nil
}
