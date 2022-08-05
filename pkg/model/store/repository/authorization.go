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

package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuthorizationRepository struct {
	collection *mongo.Collection
}

func NewAuthorizationRepository(ctx context.Context, db *mongo.Database) *AuthorizationRepository {
	collection := db.Collection("authorization")

	a := &AuthorizationRepository{collection: collection}
	_ = a.InitSchema(ctx)
	return a
}

func (a *AuthorizationRepository) InitSchema(ctx context.Context) error {
	_, err := a.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"username": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	// set admin user a default permission
	_, err = a.collection.UpdateOne(ctx,
		bson.M{"username": "admin"},
		bson.M{"$set": bson.M{"permissions": []string{"*:*:*", "*"}}},
		options.Update().SetUpsert(true),
	)
	return err
}

type Authorization struct {
	Username    string
	Permissions []string
}

func (a *AuthorizationRepository) Set(ctx context.Context, authorization *Authorization) error {
	_, err := a.collection.UpdateOne(ctx,
		bson.M{"username": authorization.Username},
		bson.M{"$set": bson.M{
			"username":    authorization.Username,
			"permissions": authorization.Permissions,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (a *AuthorizationRepository) Add(ctx context.Context, authorization *Authorization) error {
	_, err := a.collection.InsertOne(ctx, authorization)
	if err != nil {
		return err
	}
	return nil
}

func (a *AuthorizationRepository) Get(ctx context.Context, username string) (*Authorization, error) {
	authorization := &Authorization{Username: username, Permissions: []string{}}
	err := a.collection.FindOne(ctx, bson.M{"username": username}).Decode(authorization)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
		return authorization, nil
	}
	return authorization, nil
}

func (a *AuthorizationRepository) List(ctx context.Context, regexp string) ([]Authorization, error) {
	var authorizations []Authorization

	cur, err := a.collection.Find(ctx,
		bson.M{
			"permissions": bson.M{
				"$regex": "^" + regexp + "$",
			},
		},
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	if err := cur.All(ctx, &authorizations); err != nil {
		return nil, err
	}
	return authorizations, nil
}
