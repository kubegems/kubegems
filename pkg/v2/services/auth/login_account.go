// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"

	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/v2/models"
)

type AccountLoginUtil struct {
	Name string
	DB   *gorm.DB
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

	return &UserInfo{Username: user.Username, Email: user.Email}, nil
}
