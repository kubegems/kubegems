package services

import (
	restful "github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/v2/services/filters"
	"kubegems.io/kubegems/pkg/v2/services/options"
)

func enableFilters(c *restful.Container, db *gorm.DB, opts *options.Options) {
	auth := filters.NewAuthMiddleware(opts.JWT)
	audit := filters.NewAuditMiddleware()
	perms := filters.NewPermMiddleware(db)

	c.Filter(filters.Log)
	c.Filter(auth.FilterFunc)
	c.Filter(audit.FilterFunc)
	c.Filter(perms.FilterFunc)
}
