package services

import (
	restful "github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"kubegems.io/pkg/services/filters"
	"kubegems.io/pkg/services/options"
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
