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
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"kubegems.io/kubegems/pkg/model/store/repository"
)

const (
	PermissionAdmin = "*:*:*"
	PermissionNone  = ""
)

func Permission(resource, action, id string) string {
	return fmt.Sprintf("%s:%s:%s", resource, action, id)
}

type AuthorizationManager interface {
	AddPermission(ctx context.Context, username string, permissions string) error
	ListPermissions(ctx context.Context, username string) ([]string, error)
	ListUsersHasPermission(ctx context.Context, permission string) ([]string, error)
	RemovePermission(ctx context.Context, username string, permissions string) error
	HasPermission(ctx context.Context, username string, permission string) bool
}

type LocalAuthorization struct {
	repository *repository.AuthorizationRepository
}

func NewLocalAuthorization(ctx context.Context, db *mongo.Database) *LocalAuthorization {
	return &LocalAuthorization{repository: repository.NewAuthorizationRepository(ctx, db)}
}

func (a *LocalAuthorization) Init(ctx context.Context) error {
	return a.repository.InitSchema(ctx)
}

func (a *LocalAuthorization) AddPermission(ctx context.Context, username string, permission string) error {
	authorization, err := a.repository.Get(ctx, username)
	if err != nil {
		return err
	}

	exist := false

	for _, p := range authorization.Permissions {
		if p == permission {
			exist = true
			break
		}
	}

	if exist {
		return nil
	}
	authorization.Permissions = append(authorization.Permissions, permission)
	return a.repository.Set(ctx, authorization)
}

func (a *LocalAuthorization) RemovePermission(ctx context.Context, username string, permission string) error {
	authorization, err := a.repository.Get(ctx, username)
	if err != nil {
		return err
	}
	for i, p := range authorization.Permissions {
		if p == permission {
			authorization.Permissions = append(authorization.Permissions[:i], authorization.Permissions[i+1:]...)
			return a.repository.Set(ctx, authorization)
		}
	}
	return nil
}

func (a *LocalAuthorization) ListPermissions(ctx context.Context, username string) ([]string, error) {
	authorization, err := a.repository.Get(ctx, username)
	if err != nil {
		return nil, err
	}
	return authorization.Permissions, nil
}

func (a *LocalAuthorization) ListUsersHasPermission(ctx context.Context, permissionRegexp string) ([]string, error) {
	list, err := a.repository.List(ctx, permissionRegexp)
	if err != nil {
		return nil, err
	}
	users := make([]string, 0, len(list))
	for _, auth := range list {
		users = append(users, auth.Username)
	}
	return users, nil
}

func (a *LocalAuthorization) HasPermission(ctx context.Context, username string, permission string) bool {
	return true

	permissions, err := a.ListPermissions(ctx, username)
	if err != nil {
		return false
	}
	for _, p := range permissions {
		// TODO: use wildcard
		if p == permission {
			return true
		}
	}
	return false
}
