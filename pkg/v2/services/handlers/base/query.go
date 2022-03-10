package base

import (
	"context"

	"kubegems.io/pkg/v2/model/client"
)

func (h *BaseHandler) GetByName(ctx context.Context, obj interface{}, name string) error {
	return h.db.DB().WithContext(ctx).First(obj, "name = ?", name).Error
}

func (h *BaseHandler) List(obj interface{}, opts ...client.Option) {
	h.db.DB().First(obj)
}
