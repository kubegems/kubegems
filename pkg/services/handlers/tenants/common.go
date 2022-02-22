package tenanthandler

import (
	"context"

	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
)

func (h *Handler) getTenantCommon(ctx context.Context, name string) (*forms.TenantCommon, error) {
	form := &forms.TenantCommon{}
	err := h.ModelClient.Get(ctx, form.Object(), client.WhereNameEqual(name))
	return form, err
}

func (h *Handler) getTenant(ctx context.Context, name string, detail bool) (forms.FormInterface, error) {
	var form forms.FormInterface
	if detail {
		form = &forms.TenantDetail{}
	} else {
		form = &forms.TenantCommon{}
	}
	err := h.ModelClient.Get(ctx, form.Object(), client.WhereNameEqual(name))
	return form, err
}

func (h *Handler) getUserCommon(ctx context.Context, name string) (forms.UserCommon, error) {
	obj := forms.UserCommon{}
	err := h.ModelClient.Get(ctx, obj.Object(), client.WhereNameEqual(name))
	return obj, err
}

func (h *Handler) getTenantProject(ctx context.Context, tenant, name string, detail bool) (forms.FormInterface, error) {
	var form forms.FormInterface
	if detail {
		form = &forms.ProjectDetail{}
	} else {
		form = &forms.ProjectCommon{}
	}
	tenantobj := forms.TenantCommon{
		Name: tenant,
	}
	err := h.ModelClient.Get(ctx, form.Object(), client.WhereNameEqual(name), client.BelongTo(tenantobj.Object()))
	return form, err
}
