package userhandler

import (
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
)

type UserListResp struct {
	handlers.ListBase
	List []models.UserCommon `json:"list"`
}

type UserCreateResp struct {
	handlers.RespBase
	Data models.UserCreate `json:"data"`
}

type UserCommonResp struct {
	handlers.RespBase
	Data models.UserCommon `json:"data"`
}
