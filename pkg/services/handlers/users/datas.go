package userhandler

import (
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
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
