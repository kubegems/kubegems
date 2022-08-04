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
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"k8s.io/utils/pointer"
)

var InitSources = []any{
	Source{
		Name:    SourceKindHuggingface,
		BuiltIn: true,
		Online:  true,
		Enabled: true,
		Kind:    SourceKindHuggingface,
		Images:  []string{},
	},
	Source{
		Name:    SourceKindOpenMMLab,
		BuiltIn: true,
		Online:  true,
		Enabled: true,
		Kind:    SourceKindOpenMMLab,
		Images: []string{
			"kubegems/mlserver-mmlab",
		},
	},
}

type SourcesRepository struct {
	Collection *mongo.Collection
}

func NewSourcesRepository(db *mongo.Database) *SourcesRepository {
	collection := db.Collection("sources")
	return &SourcesRepository{collection}
}

func (r *SourcesRepository) InitSchema(ctx context.Context) error {
	const uniqueNameIndex = "unique_name"
	if _, err := r.Collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
		},
		Options: &options.IndexOptions{
			Unique: pointer.Bool(true),
			Name:   pointer.String(uniqueNameIndex),
		},
	}); err != nil {
		return err
	}

	counts, err := r.Collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if counts == 0 {
		_, err := r.Collection.InsertMany(ctx, InitSources)
		if err != nil {
			return err
		}
	}
	return err
}

type GetSourceOptions struct {
	WithDisabled bool
	WithCounts   bool
	WithAuth     bool
}

func (r *SourcesRepository) withModelCountsStage() []bson.D {
	return []bson.D{
		{{
			Key: "$lookup", Value: bson.M{
				"from": "models",
				"let":  bson.M{"name": "$name"},
				"pipeline": bson.A{
					bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$source", "$$name"}}}},
					bson.M{"$group": bson.M{
						"_id":   "$source",
						"total": bson.M{"$sum": 1},
					}},
				},
				"as": "modelsCount",
			},
		}},
		{{
			Key: "$set", Value: bson.M{
				"modelsCount": bson.M{
					"$getField": bson.M{
						"input": bson.M{
							"$arrayElemAt": bson.A{"$modelsCount", 0},
						},
						"field": "total",
					},
				},
			},
		}},
	}
}

func (r *SourcesRepository) Get(ctx context.Context, name string, opts GetSourceOptions) (*SourceWithAddtional, error) {
	cond := bson.M{"name": name}
	if !opts.WithDisabled {
		cond["enabled"] = true
	}

	pipline := mongo.Pipeline{
		{{Key: "$match", Value: cond}},
	}
	if opts.WithCounts {
		pipline = append(pipline, r.withModelCountsStage()...)
	}
	if !opts.WithAuth {
		pipline = append(pipline, bson.D{{Key: "$unset", Value: bson.A{"auth"}}})
	}
	cur, err := r.Collection.Aggregate(ctx, pipline)
	if err != nil {
		return nil, err
	}
	var list []SourceWithAddtional
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return &list[0], nil
	}
	return nil, mongo.ErrNoDocuments
}

type ListSourceOptions struct {
	WithDisabled    bool
	WithModelCounts bool
	WithAuth        bool
}

func (o ListSourceOptions) ToConditionAndFindOptions() (interface{}, *options.FindOptions) {
	cond := bson.M{}
	if !o.WithDisabled {
		cond["enabled"] = true
	}
	return cond, options.Find().SetSort(bson.M{"name": 1})
}

type SourceWithAddtional struct {
	Source     `bson:",inline" json:",inline"`
	ModelCount int64 `bson:"modelsCount" json:"modelsCount"`
}

func (r *SourcesRepository) List(ctx context.Context, opts ListSourceOptions) ([]SourceWithAddtional, error) {
	cond, findopts := opts.ToConditionAndFindOptions()

	var cursor *mongo.Cursor
	var err error

	pipline := mongo.Pipeline{
		{{Key: "$match", Value: cond}},
		{{Key: "$sort", Value: findopts.Sort}},
	}
	if opts.WithModelCounts {
		pipline = append(pipline, r.withModelCountsStage()...)
	}
	if !opts.WithAuth {
		pipline = append(pipline, bson.D{{Key: "$unset", Value: bson.A{"auth"}}})
	}
	cursor, err = r.Collection.Aggregate(ctx, pipline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	sources := []SourceWithAddtional{}
	if err := cursor.All(ctx, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *SourcesRepository) Count(ctx context.Context, opts ListSourceOptions) (int64, error) {
	cond, _ := opts.ToConditionAndFindOptions()
	return r.Collection.CountDocuments(ctx, cond)
}

func (s *SourcesRepository) Create(ctx context.Context, source *Source) error {
	now := time.Now()
	if source.CreationTime.IsZero() {
		source.CreationTime = now
	}
	source.UpdationTime = now
	result, err := s.Collection.InsertOne(ctx, source)
	if err != nil {
		return err
	}
	switch val := result.InsertedID.(type) {
	case string:
		source.ID = val
	case primitive.ObjectID:
		source.ID = val.Hex()
	}
	return nil
}

func (s *SourcesRepository) Update(ctx context.Context, source *Source) error {
	now := time.Now()
	source.UpdationTime = now

	_, err := s.Collection.UpdateOne(ctx,
		bson.D{
			{Key: "name", Value: source.Name},
		},
		bson.M{"$set": bson.D{
			{Key: "builtin", Value: source.BuiltIn},
			{Key: "online", Value: source.Online},
			{Key: "updationtime", Value: now},
			{Key: "images", Value: source.Images},
			{Key: "enabled", Value: source.Enabled},
			{Key: "address", Value: source.Address},
			{Key: "kind", Value: source.Kind},
			{Key: "annotations", Value: source.Annotations},
		}},
	)
	return err
}

func (r SourcesRepository) Delete(ctx context.Context, source *Source) error {
	result := r.Collection.FindOneAndDelete(ctx, bson.M{"name": source.Name})
	if err := result.Err(); err != nil {
		return err
	}
	return result.Decode(source)
}
