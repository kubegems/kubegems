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

type SourcesRepository struct {
	Collection *mongo.Collection
}

func NewSourcesRepository(db *mongo.Database) *SourcesRepository {
	collection := db.Collection("sources")
	return &SourcesRepository{collection}
}

func (r *SourcesRepository) InitSchema(ctx context.Context) error {
	const uniqueNameIndex = "unique_name"
	_, err := r.Collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
		},
		Options: &options.IndexOptions{
			Unique: pointer.Bool(true),
			Name:   pointer.String(uniqueNameIndex),
		},
	})
	return err
}

func (r *SourcesRepository) Get(ctx context.Context, name string) (*Source, error) {
	var source Source
	err := r.Collection.FindOne(ctx, bson.M{"name": name}).Decode(&source)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

type ListSourceOptions struct {
	CommonListOptions
}

func (o ListSourceOptions) ToConditionAndFindOptions() (interface{}, *options.FindOptions) {
	cond := bson.M{}
	if o.Search != "" {
		cond["name"] = bson.M{"$regex": o.Search}
	}
	if o.Page <= 0 {
		o.Page = 1
	}
	if o.Size <= 0 {
		o.Size = 10
	}
	return cond, options.Find().SetSort(bson.M{"name": 1}).SetLimit(o.Size).SetSkip((o.Page - 1) * o.Size)
}

func (r *SourcesRepository) List(ctx context.Context, opts ListSourceOptions) ([]Source, error) {
	cond, mongoopts := opts.ToConditionAndFindOptions()
	cursor, err := r.Collection.Find(ctx, cond, mongoopts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	sources := []Source{}
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

func (r SourcesRepository) Delete(ctx context.Context, name string) error {
	_, err := r.Collection.DeleteOne(ctx, bson.M{"name": name})
	return err
}
