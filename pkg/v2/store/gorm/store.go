package gormstore

import (
	"context"

	"kubegems.io/kubegems/pkg/v2/store"
)

type GORMStore struct{}

func (s *GORMStore) List(context.Context, store.EntityObjectList, ...store.ListOptions) error
func (s *GORMStore) Create(context.Context, store.EntityObject, ...store.CreateOption) error
func (s *GORMStore) CreateMany(context.Context, store.EntityObjectList, ...store.CreateOption) error
func (s *GORMStore) Retrieve(context.Context, store.EntityObject, ...store.RetrieveOptions) error
func (s *GORMStore) Update(context.Context, store.EntityObject, ...store.UpdateOptions) error
func (s *GORMStore) Delete(context.Context, store.EntityObject, store.DeleteOptions) error
