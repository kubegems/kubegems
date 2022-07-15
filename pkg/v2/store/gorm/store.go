// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
