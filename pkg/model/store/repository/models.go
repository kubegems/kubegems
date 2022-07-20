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
	const source_name_index = "source_name_index"
	_, err := m.Collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "source", Value: 1},
			{Key: "name", Value: 1},
		},
		Options: &options.IndexOptions{
			Name:   pointer.String(source_name_index),
			Unique: pointer.Bool(true),
		},
	})
	return err
}

type ModelListOptions struct {
	CommonListOptions
	Source     string
	Tags       []string
	License    string
	Framework  string
	Task       string
	WithRating bool
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
	Model    `bson:",inline" json:",inline"`
	Versions []string `bson:"versions" json:"versions"`
	Rating   Rating   `bson:"rating" json:"rating"`
}

func (m *ModelsRepository) Get(ctx context.Context, source, name string) (ModelWithAddtional, error) {
	cond := bson.M{"source": source, "name": name}
	ret := ModelWithAddtional{}
	err := m.Collection.FindOne(ctx, cond).Decode(&ret)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ret, response.NewError(http.StatusNotFound, fmt.Sprintf("model %s not found", name))
		}
		return ModelWithAddtional{}, err
	}
	// set default version
	if len(ret.Versions) == 0 {
		ret.Versions = []string{"latest"}
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
	var err error
	var cursor *mongo.Cursor
	if opts.WithRating {
		aggregate := []bson.M{
			{"$match": cond},
			{"$sort": findoptions.Sort},
			{"$skip": findoptions.Skip},
			{"$limit": findoptions.Limit},
			{"$lookup": bson.M{
				"from": "comments",
				"let": bson.M{
					"postid": bson.M{"$concat": []string{"$source", "/", "$name"}},
				},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": []string{"$postid", "$$postid"},
							},
							"rating": bson.M{"$gt": 0},
						},
					},
					{
						"$group": bson.M{
							"_id":    "$postid",
							"rating": bson.M{"$avg": "$rating"},
							"count":  bson.M{"$sum": 1},
							"total":  bson.M{"$sum": "$rating"},
						},
					},
				},
				"as": "ratings",
			}},
			{
				"$project": bson.M{
					"_id":          0,
					"source":       1,
					"name":         1,
					"ratings":      1,
					"framework":    1,
					"likes":        1,
					"task":         1,
					"recomment":    1,
					"downloads":    1,
					"create_at":    bson.M{"$toString": "$create_at"},
					"update_at":    bson.M{"$toString": "$update_at"},
					"lastModified": bson.M{"$toString": "$lastModified"},
				},
			},
		}
		cursor, err = m.Collection.Aggregate(ctx, aggregate, options.Aggregate().SetAllowDiskUse(true))
	} else {
		cursor, err = m.Collection.Find(ctx, cond, findoptions)
	}

	if err != nil {
		return nil, err
	}

	result := []struct {
		ModelWithAddtional `bson:",inline" json:",inline"`
		Ratings            []Rating `bson:"ratings" json:"ratings"`
	}{}
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	ret := make([]ModelWithAddtional, len(result))
	for i, item := range result {
		ret[i] = item.ModelWithAddtional
		if len(item.Ratings) > 0 {
			ret[i].Rating = item.Ratings[0]
		}
	}
	return ret, nil
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
				"intro":     model.Intro,
				"recomment": model.Recomment,
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
