package gormstore

import (
	"context"
	"fmt"

	"kubegems.io/kubegems/pkg/apis/gems/v1beta1"
	"kubegems.io/kubegems/pkg/v2/store"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeStore struct {
	Converter
	client.Client
}

func (s *KubeStore) List(ctx context.Context, list store.EntityObjectList, opts ...store.ListOptions) error {
	ol, err := s.EntityList2ObjectList(list)
	if err != nil {
		return err
	}
	if err := s.Client.List(ctx, ol, &client.ListOptions{}); err != nil {
		return err
	}
	if err := s.ObjectList2EntityList(ol, list); err != nil {
		return err
	}
	return nil

}
func (s *KubeStore) Create(context.Context, store.EntityObject, ...store.CreateOption) error
func (s *KubeStore) CreateMany(context.Context, store.EntityObjectList, ...store.CreateOption) error
func (s *KubeStore) Retrieve(context.Context, store.EntityObject, ...store.RetrieveOptions) error
func (s *KubeStore) Update(context.Context, store.EntityObject, ...store.UpdateOptions) error
func (s *KubeStore) Delete(context.Context, store.EntityObject, store.DeleteOptions) error

type Converter struct{}

func (c *Converter) EntityList2ObjectList(el store.EntityObjectList) (client.ObjectList, error) {
	switch el.Kind() {
	case "environments":
		return &v1beta1.EnvironmentList{}, nil
	case "tenants":
		return &v1beta1.TenantList{}, nil
	default:
		return nil, fmt.Errorf("unsupport kind %s for store %s", el.Kind(), "KubeStore")
	}
}

func (c *Converter) ObjectList2EntityList(ol client.ObjectList, el store.EntityObjectList) error {
	switch objList := ol.(type) {
	case *v1beta1.TenantList:
		fmt.Println(objList)
	case *v1beta1.EnvironmentList:
		fmt.Println(objList)
	}
	return nil
}
