package base

import (
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/aaa/audit"
	"kubegems.io/pkg/service/aaa/authorization"
)

// BaseHandler is the base handler for all handlers
type BaseHandler struct {
	audit.AuditInterface
	authorization.PermissionChecker
	aaa.UserInterface
}

func NewBaseHandler(audit audit.AuditInterface, permission authorization.PermissionChecker, user aaa.UserInterface) *BaseHandler {
	return &BaseHandler{
		AuditInterface:    audit,
		PermissionChecker: permission,
		UserInterface:     user,
	}
}
