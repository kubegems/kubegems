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
	collection := db.Collection("test1")
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
	Source    string
	Tags      []string
	Framework string
}

func (o *ModelListOptions) ToConditionAndFindOptions() (interface{}, *options.FindOptions) {
	cond := bson.M{}
	if o.Source != "" {
		cond["source"] = o.Source
	}
	if o.Search != "" {
		cond["$text"] = bson.M{"$search": o.Search}
	}
	if len(o.Tags) != 0 {
		cond["tags"] = bson.M{"$all": o.Tags}
	}
	if o.Framework != "" {
		cond["framework"] = o.Framework
	}

	sort := bson.M{}
	for _, item := range strings.Split(o.Sort, ",") {
		if item == "" {
			continue
		}
		if item[0] == '-' {
			sort[item[1:]] = -1
		} else {
			sort[item] = 1
		}
	}

	if o.Page <= 0 {
		o.Page = 1
	}
	if o.Size <= 0 {
		o.Size = 10
	}
	return cond, options.Find().SetSort(sort).SetLimit(o.Size).SetSkip((o.Page - 1) * o.Size)
}

func (m *ModelsRepository) Get(ctx context.Context, source, name string) (Model, error) {
	cond := bson.M{"source": source, "name": name}
	ret := Model{}
	err := m.Collection.FindOne(ctx, cond).Decode(&ret)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ret, response.NewError(http.StatusNotFound, fmt.Sprintf("model %s not found", name))
		}
		return Model{}, err
	}
	return ret, nil
}

func (m *ModelsRepository) Count(ctx context.Context, opts ModelListOptions) (int64, error) {
	cond, _ := opts.ToConditionAndFindOptions()
	return m.Collection.CountDocuments(ctx, cond)
}

func (m *ModelsRepository) List(ctx context.Context, opts ModelListOptions) ([]Model, error) {
	cond, options := opts.ToConditionAndFindOptions()
	cur, err := m.Collection.Find(ctx, cond, options)
	if err != nil {
		return nil, err
	}
	result := []Model{}
	err = cur.All(ctx, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *ModelsRepository) Create(ctx context.Context, model Model) error {
	_, err := m.Collection.InsertOne(ctx, model)
	return err
}

func (m *ModelsRepository) Delete(ctx context.Context, source, name string) error {
	_, err := m.Collection.DeleteOne(ctx, bson.M{"source": source, "name": name})
	return err
}

type Selectors struct {
	Tags      []string `json:"tags"`
	Libraries []string `json:"libraries"`
	Licenses  []string `json:"licenses"`
}

func (m *ModelsRepository) ListSelectors(ctx context.Context, listopts ModelListOptions) (*Selectors, error) {
	cond, _ := listopts.ToConditionAndFindOptions()
	distincttags, _ := m.Collection.Distinct(ctx, "tags", cond)
	distinctlibraries, _ := m.Collection.Distinct(ctx, "library", cond)
	distinctlicenses, _ := m.Collection.Distinct(ctx, "license", cond)

	tostrings := func(data []interface{}) []string {
		ret := make([]string, 0, len(data))
		for _, item := range data {
			switch val := item.(type) {
			case string:
				ret = append(ret, val)
			default:
				continue
			}
		}
		return ret
	}

	selectors := &Selectors{
		Tags:      tostrings(distincttags),
		Libraries: tostrings(distinctlibraries),
		Licenses:  tostrings(distinctlicenses),
	}
	return selectors, nil
}
