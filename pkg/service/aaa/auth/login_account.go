package auth

import (
	"context"
	"time"

	"gorm.io/gorm"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
)

type AccountLoginUtil struct {
	Name string
	DB   *gorm.DB
}

func (ut *AccountLoginUtil) GetName() string {
	return ut.Name
}

func (ut *AccountLoginUtil) LoginAddr() string {
	return DefaultLoginURL
}

func (ut *AccountLoginUtil) GetUserInfo(ctx context.Context, cred *Credential) (*UserInfo, error) {
	user := &models.User{}
	if err := ut.DB.WithContext(ctx).Where("username = ?", cred.Username).First(user).Error; err != nil {
		return nil, err
	}
	if err := utils.ValidatePassword(cred.Password, user.Password); err != nil {
		return nil, err
	}
	now := time.Now()
	user.LastLoginAt = &now

	return &UserInfo{Username: user.Username, Email: user.Email}, nil
}
