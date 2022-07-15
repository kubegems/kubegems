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

package client

import (
	"context"
)

type HookPhase string

var (
	BeforeUpdate HookPhase = "BeforeUpdate"
	BeforeCreate HookPhase = "BeforeCreate"
	BeforeDelete HookPhase = "BeforeDelete"

	AfterUpdate HookPhase = "AfterUpdate"
	AfterCreate HookPhase = "AfterCreate"
	AfterDelete HookPhase = "AfterDelete"
)

type ModelClientIface interface {
	Create(ctx context.Context, obj Object, opts ...Option) error
	Get(ctx context.Context, obj Object, opts ...Option) error
	Update(ctx context.Context, obj Object, opts ...Option) error
	Delete(ctx context.Context, obj Object, opts ...Option) error
	CreateInBatches(ctx context.Context, objs ObjectListIface, opts ...Option) error
	List(ctx context.Context, olist ObjectListIface, opts ...Option) error
	Count(ctx context.Context, o ObjectTypeIface, t *int64, opts ...Option) error
}
