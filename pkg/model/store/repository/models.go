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
	"fmt"
	"net/http"
	"strings"

	"github.com/goharbor/harbor/src/lib/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type ModelsRepository struct {
	Collection *mongo.Collection
}

func NewModelsRepository(db *mongo.Database) *ModelsRepository {
	collection := db.Collection("models")
	return &ModelsRepository{Collection: collection}
}

func (m *ModelsRepository) InitSchema(ctx context.Context) error {
	names, err := m.Collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "source", Value: 1}, {Key: "name", Value: 1}},
			Options: &options.IndexOptions{Unique: pointer.Bool(true)},
		},
		// we used this unio index at list models page
		{Keys: bson.D{
			{Key: "recomment", Value: -1},
			{Key: "downloads", Value: -1},
			{Key: "name", Value: 1},
		}},
	})
	_ = names
	return err
}

type ModelListOptions struct {
	CommonListOptions
	Source       string
	Tags         []string
	License      string
	Framework    string
	Task         string
	WithRating   bool
	WithDisabled bool
	WithVersions bool
}

func (o *ModelListOptions) ToConditionAndFindOptions() (interface{}, *options.FindOptions) {
	cond := bson.M{}
	if o.Source != "" {
		cond["source"] = o.Source
	}
	if o.Search != "" {
		cond["name"] = bson.M{"$regex": o.Search}
	}
	if len(o.Tags) != 0 {
		cond["tags"] = bson.M{"$all": o.Tags}
	}
	if o.Framework != "" {
		cond["framework"] = o.Framework
	}
	if o.License != "" {
		cond["license"] = o.License
	}
	if o.Task != "" {
		cond["task"] = o.Task
	}
	if !o.WithDisabled {
		cond["enabled"] = true
	}

	sort := bson.D{}

	if o.Sort != "" {
		for _, item := range strings.Split(o.Sort, ",") {
			if item == "" {
				continue
			}
			if item[0] == '-' {
				sort = append(sort, bson.E{Key: item[1:], Value: -1})
			} else {
				sort = append(sort, bson.E{Key: item, Value: 1})
			}
		}
	} else {
		// 默认排序以 推荐值 降序，名称升序
		sort = append(sort, bson.E{Key: "recomment", Value: -1})
		sort = append(sort, bson.E{Key: "downloads", Value: -1})
		sort = append(sort, bson.E{Key: "name", Value: 1})
	}

	if o.Page <= 0 {
		o.Page = 1
	}
	if o.Size <= 0 {
		o.Size = 10
	}
	return cond, options.Find().SetSort(sort).SetLimit(o.Size).SetSkip((o.Page - 1) * o.Size)
}

type ModelWithAddtional struct {
	Model  `bson:",inline" json:",inline"`
	Rating *Rating `bson:"rating" json:"rating"`
}

func (m *ModelsRepository) Get(ctx context.Context, source, name string, includedisabled bool) (ModelWithAddtional, error) {
	cond := bson.M{"source": source, "name": name}
	if !includedisabled {
		cond["enabled"] = true
	}

	ret := ModelWithAddtional{}
	if err := m.Collection.FindOne(ctx, cond).Decode(&ret); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ret, response.NewError(http.StatusNotFound, fmt.Sprintf("model %s not found", name))
		}
		return ModelWithAddtional{}, err
	}
	return ret, nil
}

func (m *ModelsRepository) Count(ctx context.Context, opts ModelListOptions) (int64, error) {
	cond, _ := opts.ToConditionAndFindOptions()
	return m.Collection.CountDocuments(ctx, cond)
}

// nolint: funlen
func (m *ModelsRepository) List(ctx context.Context, opts ModelListOptions) ([]ModelWithAddtional, error) {
	cond, findoptions := opts.ToConditionAndFindOptions()
	showfields := bson.M{
		"_id":              0,
		"source":           1,
		"name":             1,
		"rating":           1,
		"framework":        1,
		"likes":            1,
		"task":             1,
		"recomment":        1,
		"recommentcontent": 1,
		"downloads":        1,
		"tags":             1,
		"created_at":       1,
		"updated_at":       1,
		"lastModified":     1,
		"enabled":          1,
	}

	pipline := []bson.M{
		{"$match": cond},
		{"$sort": findoptions.Sort},
		{"$skip": findoptions.Skip},
		{"$limit": findoptions.Limit},
	}
	if opts.WithRating {
		pipline = append(pipline,
			bson.M{
				"$lookup": bson.M{
					"from": "comments",
					"let":  bson.M{"postid": bson.M{"$concat": []string{"$source", "/", "$name"}}},
					"pipeline": []bson.M{
						{"$match": bson.M{
							"$expr":  bson.M{"$eq": []string{"$postid", "$$postid"}},
							"rating": bson.M{"$gt": 0},
						}},
						{"$group": bson.M{
							"_id":    "$postid",
							"rating": bson.M{"$avg": "$rating"},
							"count":  bson.M{"$sum": 1},
							"total":  bson.M{"$sum": "$rating"},
						}},
					},
					"as": "rating",
				},
			},
			bson.M{
				"$set": bson.M{"rating": bson.M{"$arrayElemAt": bson.A{"$rating", 0}}},
			},
		)
		showfields["rating"] = 1
	}
	if opts.WithVersions {
		// set default versions, cause we do not have any other version
		pipline = append(pipline, bson.M{
			"$set": bson.M{"versions": bson.A{"latest"}},
		})
		showfields["versions"] = 1
	}
	if opts.WithDisabled {
		showfields["enabled"] = 1
	}

	pipline = append(pipline, bson.M{"$project": showfields})
	cursor, err := m.Collection.Aggregate(ctx, pipline)
	if err != nil {
		return nil, err
	}
	into := []ModelWithAddtional{}
	if err = cursor.All(ctx, &into); err != nil {
		return nil, err
	}
	return into, nil
}

func (m *ModelsRepository) Create(ctx context.Context, model Model) error {
	_, err := m.Collection.InsertOne(ctx, model)
	return err
}

func (m *ModelsRepository) Update(ctx context.Context, model *Model) error {
	result := m.Collection.FindOneAndUpdate(ctx,
		bson.M{"source": model.Source, "name": model.Name},
		bson.M{
			"$set": bson.M{
				"recomment":        model.Recomment,
				"recommentcontent": model.RecommentContent,
				"tags":             model.Tags,
				"enabled":          model.Enabled,
			},
		},
	)
	if err := result.Err(); err != nil {
		return err
	}
	_ = result.Decode(model)
	return nil
}

func (m *ModelsRepository) Delete(ctx context.Context, source, name string) error {
	_, err := m.Collection.DeleteOne(ctx, bson.M{"source": source, "name": name})
	return err
}

type Selectors struct {
	Tags       []string `json:"tags"`
	Frameworks []string `json:"frameworks"`
	Licenses   []string `json:"licenses"`
	Tasks      []string `json:"tasks"`
}

func (m *ModelsRepository) ListSelectors(ctx context.Context, listopts ModelListOptions) (*Selectors, error) {
	cond, _ := listopts.ToConditionAndFindOptions()
	distincttags, _ := m.Collection.Distinct(ctx, "tags", cond)
	distinctframeworks, _ := m.Collection.Distinct(ctx, "framework", cond)
	distinctlicenses, _ := m.Collection.Distinct(ctx, "license", cond)
	distincttasks, _ := m.Collection.Distinct(ctx, "task", cond)

	tostrings := func(data []interface{}) []string {
		ret := make([]string, 0, len(data))
		for _, item := range data {
			if val, ok := item.(string); ok && val != "" {
				ret = append(ret, val)
			}
		}
		return ret
	}
	selectors := &Selectors{
		Tags:       tostrings(distincttags),
		Frameworks: tostrings(distinctframeworks),
		Licenses:   tostrings(distinctlicenses),
		Tasks:      tostrings(distincttasks),
	}
	return selectors, nil
}

func (r *ModelsRepository) ListVersions(ctx context.Context, source, model string) ([]ModelVersion, error) {
	cond := bson.M{
		"source": source,
		"name":   model,
	}
	opts := &options.FindOneOptions{
		Projection: bson.M{"versions": 1}, // only return versions

	}
	modelWithVersion := &Model{}
	if err := r.Collection.FindOne(ctx, cond, opts).Decode(modelWithVersion); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("model %s/%s not found", source, model)
		}
		return nil, err
	}
	return modelWithVersion.Versions, nil
}

func (r *ModelsRepository) GetVersion(ctx context.Context, source, model, version string) (ModelVersion, error) {
	versions, err := r.ListVersions(ctx, source, model)
	if err != nil {
		return ModelVersion{}, err
	}
	for _, v := range versions {
		if v.Name == version {
			return v, nil
		}
	}
	return ModelVersion{}, fmt.Errorf("version %s not found", version)
}
