package userhandler

import (
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
)

type UserListResp struct {
	handlers.ListBase
	List []models.User `json:"list"`
}
