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

package client

import "time"

type SystemRole struct {
	ID       uint
	RoleName string
	RoleCode string
}

type CommonUser struct {
	ID           uint
	Username     string
	Email        string
	Phone        string
	Password     string
	IsActive     *bool
	Kind         string
	Source       string
	CreatedAt    *time.Time
	LastLoginAt  *time.Time
	SystemRole   *SystemRole
	SystemRoleID uint
}

func (u *CommonUser) GetID() uint {
	return u.ID
}

func (u *CommonUser) GetSystemRoleID() uint {
	return u.SystemRoleID
}

func (u *CommonUser) GetUsername() string {
	return u.Username
}

func (u *CommonUser) GetKind() string {
	return u.Kind
}

func (u *CommonUser) GetEmail() string {
	return u.Email
}

func (u *CommonUser) GetSource() string {
	return u.Source
}
