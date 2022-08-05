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
)

type CommentsRepository struct {
	Collection *mongo.Collection
}

func NewCommentsRepository(db *mongo.Database) *CommentsRepository {
	collection := db.Collection("comments")
	return &CommentsRepository{Collection: collection}
}

func (c *CommentsRepository) InitSchema(ctx context.Context) error {
	// add index on replyto.rootid
	_, err := c.Collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"replyto.rootid": 1},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *CommentsRepository) Create(ctx context.Context, postID string, comment *Comment) error {
	now := time.Now()
	if comment.CreationTime.IsZero() {
		comment.CreationTime = now
	}
	comment.ID = "" // clear id
	comment.UpdationTime = now
	comment.PostID = postID
	result, err := c.Collection.InsertOne(ctx, comment)
	if err != nil {
		return err
	}
	switch val := result.InsertedID.(type) {
	case string:
		comment.ID = val
	case primitive.ObjectID:
		comment.ID = val.Hex()
	}
	return nil
}

func (c *CommentsRepository) Update(ctx context.Context, comment *Comment) error {
	id, err := primitive.ObjectIDFromHex(comment.ID)
	if err != nil {
		return err
	}
	_, err = c.Collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"updationtime": time.Now(),
				"content":      comment.Content,
				"rating":       comment.Rating,
			},
		})
	return err
}

func (c *CommentsRepository) Delete(ctx context.Context, comment *Comment) error {
	id, err := primitive.ObjectIDFromHex(comment.ID)
	if err != nil {
		return err
	}
	result, err := c.Collection.DeleteOne(ctx, bson.M{
		"_id":      id,
		"username": comment.Username,
	})
	_ = result.DeletedCount
	return err
}

type ListCommentOptions struct {
	CommonListOptions
	PostID      string // find comments of this post
	ReplyToID   string // find all replies of this comment
	WithReplies bool   // include replies in the result
}

func (o ListCommentOptions) ToConditionAndFindOptions() (interface{}, *options.FindOptions) {
	condition := bson.M{
		"postid": o.PostID,
	}
	if o.ReplyToID != "" {
		condition["replyto.rootid"] = o.ReplyToID
	} else {
		condition["replyto.rootid"] = bson.M{"$exists": false}
	}

	findOptions := options.Find().SetSort(bson.M{"creationtime": -1})
	if o.Size > 0 {
		findOptions.SetLimit(o.Size)
	}
	if o.Page > 0 {
		findOptions.SetSkip(o.Size * (o.Page - 1))
	}
	return condition, findOptions
}

type CommentWithAddtional struct {
	Comment      `json:",inline" bson:",inline"`
	RepliesCount int64                  `json:"repliesCount,omitempty"`
	Replies      []CommentWithAddtional `json:"replies,omitempty"`
}

func (c *CommentsRepository) List(ctx context.Context, listoptions ListCommentOptions) ([]CommentWithAddtional, error) {
	cond, findopts := listoptions.ToConditionAndFindOptions()

	var cur *mongo.Cursor
	var err error

	if listoptions.WithReplies {
		aggregate := []bson.M{
			{"$match": cond},
			{"$sort": findopts.Sort},
			{"$skip": findopts.Skip},
			{"$limit": findopts.Limit},
			{"$lookup": bson.M{
				"from": "comments",
				"let": bson.M{
					"id": bson.M{"$toString": "$_id"},
				},
				"pipeline": bson.A{
					bson.M{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": bson.A{"$replyto.rootid", "$$id"},
							},
						},
					},
				},
				"as": "replies",
			}},
			{"$set": bson.M{"repliescount": bson.M{"$size": "$replies"}}},
		}
		cur, err = c.Collection.Aggregate(ctx, aggregate, options.Aggregate())
	} else {
		// just find
		cur, err = c.Collection.Find(ctx, cond, findopts)
	}
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	comments := []CommentWithAddtional{}
	if err := cur.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

func (c *CommentsRepository) Count(ctx context.Context, listoptions ListCommentOptions) (int64, error) {
	cond, _ := listoptions.ToConditionAndFindOptions()
	count, err := c.Collection.CountDocuments(ctx, cond)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// nolint: tagliatelle
type Rating struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Rating float64 `json:"rating"`
	Count  int64   `json:"count"`
	Total  int64   `json:"total"`
}

func (c *CommentsRepository) Rating(ctx context.Context, ids ...string) ([]Rating, error) {
	cur, err := c.Collection.Aggregate(ctx, bson.A{
		bson.M{
			"$match": bson.M{
				"postid": bson.M{
					"$in": ids,
				},
				"rating": bson.M{"$gt": 0},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id": "$postid",
				"rating": bson.M{
					"$avg": "$rating",
				},
				"count": bson.M{
					"$sum": 1,
				},
				"total": bson.M{
					"$sum": "$rating",
				},
			},
		},
	},
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	results := []Rating{}
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
