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
