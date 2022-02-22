package auth

import (
	"context"

	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/utils"
)

type AccountLoginUtil struct {
	Name        string
	ModelClient client.ModelClientIface
}

func (ut *AccountLoginUtil) LoginAddr() string {
	return DefaultLoginURL
}

func (ut *AccountLoginUtil) GetUserInfo(ctx context.Context, cred *Credential) (*UserInfo, error) {
	user := forms.UserInternal{}
	if err := ut.ModelClient.Get(ctx, user.Object(), client.Where("username", client.Eq, cred.Username)); err != nil {
		return nil, err
	}
	if err := utils.ValidatePassword(cred.Password, user.Password); err != nil {
		return nil, err
	}

	return &UserInfo{Username: user.Name, Email: user.Email}, nil
}
