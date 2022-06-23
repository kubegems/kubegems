package repository

import (
	"context"
	"time"

	"github.com/emicklei/go-restful/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"kubegems.io/kubegems/pkg/model/store/types"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

type Comments struct {
	Collection *mongo.Collection
}

func (c *Comments) Create(ctx context.Context, postID string, comment *types.Comment) error {
	now := time.Now()
	if comment.CreationTime.IsZero() {
		comment.CreationTime = now
	}
	if comment.UpdationTime.IsZero() {
		comment.UpdationTime = now
	}

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

type ListCommentOptions struct {
	Page    int64  `json:"page,omitempty"`
	Size    int64  `json:"size,omitempty"`
	ReplyID string `json:"replyID,omitempty"` // find comments reply to this comment
}

func (c *Comments) List(ctx context.Context, postID string, listoptions ListCommentOptions) ([]*types.Comment, error) {
	if listoptions.Page == 0 {
		listoptions.Page = 1
	}
	offset := (listoptions.Page - 1) * listoptions.Size

	var filter interface{}
	if listoptions.ReplyID == "" {
		filter = bson.D{
			{Key: "postid", Value: postID},
			{Key: "deleted", Value: false},
			{Key: "replyid", Value: bson.M{"$exists": false}},
		}
	} else {
		filter = bson.D{
			{Key: "postid", Value: postID},
			{Key: "deleted", Value: false},
			{Key: "replyid", Value: listoptions.ReplyID},
		}
	}
	cur, err := c.Collection.Find(ctx, filter,
		&options.FindOptions{
			Skip:  &offset,
			Limit: &listoptions.Size,
			Sort: bson.D{
				{Key: "creationtime", Value: -1},
			},
		})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	comments := []*types.Comment{}
	if err := cur.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// nolint: tagliatelle
type Rating struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Rating float64 `json:"rating,omitempty"`
	Count  int64   `json:"count,omitempty"`
	Total  int64   `json:"total,omitempty"`
}

func (c *Comments) Rating(ctx context.Context, postID string) (Rating, error) {
	rating := Rating{
		ID: postID,
	}
	cur, err := c.Collection.Aggregate(ctx, bson.A{
		bson.M{
			"$match": bson.M{
				"postid": postID,
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
		return rating, err
	}
	defer cur.Close(ctx)

	results := []Rating{}
	if err := cur.All(ctx, &results); err != nil {
		return rating, err
	}
	// no rating
	if len(results) == 0 {
		return rating, nil
	}
	return results[0], nil
}

// nolint: gomnd,funlen
func (c *Comments) AddToWebservice(ws *restful.WebService) error {
	getpostid := func(req *restful.Request) string {
		registry := req.PathParameter("registry")
		model := req.PathParameter("model")
		return registry + "/" + model
	}

	listmodels := func(r *restful.Request, w *restful.Response) {
		listoptions := ListCommentOptions{
			Page:    request.Query(r.Request, "page", int64(1)),
			Size:    request.Query(r.Request, "size", int64(10)),
			ReplyID: request.Query(r.Request, "replyID", ""),
		}
		comments, err := c.List(r.Request.Context(), getpostid(r), listoptions)
		if err != nil {
			response.ServerError(w, err)
			return
		}
		response.OK(w, response.Page{
			List: comments,
			Page: listoptions.Page,
			Size: listoptions.Size,
		})
	}

	postComment := func(r *restful.Request, w *restful.Response) {
		comment := &types.Comment{}
		if err := request.Body(r.Request, comment); err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		if err := c.Create(r.Request.Context(), getpostid(r), comment); err != nil {
			response.ServerError(w, err)
			return
		}
		response.OK(w, comment)
	}

	rating := func(r *restful.Request, w *restful.Response) {
		avgrating, err := c.Rating(r.Request.Context(), getpostid(r))
		if err != nil {
			response.ServerError(w, err)
			return
		}
		response.OK(w, avgrating)
	}

	tree := &route.Tree{
		// add response wrapper
		RouteUpdateFunc: func(r *route.Route) {
			paged := false
			for _, item := range r.Params {
				if item.Kind == route.ParamKindQuery && item.Name == "page" {
					paged = true
					break
				}
			}
			for i, v := range r.Responses {
				//  if query parameters exist, response as a paged response
				if paged {
					r.Responses[i].Body = response.Response{Data: response.Page{List: v.Body}}
				} else {
					r.Responses[i].Body = response.Response{Data: v.Body}
				}
			}
		},
		Group: route.
			NewGroup("/registries/{registry}/models/{model}").
			Parameters(
				route.PathParameter("registry", "registry name"),
				route.PathParameter("model", "model name"),
			).
			AddSubGroup(
				route.NewGroup("/comments").Tag("comments").
					AddRoutes(
						route.GET("").To(listmodels).
							Paged().
							Parameters(
								route.QueryParameter("replyID", "list comments reply to this comment").Optional(),
							).
							Response([]types.Comment{}).
							ShortDesc("List comments for a model"),
						route.POST("").To(postComment).
							Parameters(
								route.BodyParameter("comment", types.Comment{}),
							).
							Response(types.Comment{}).
							ShortDesc("Create a comment"),
					),
				route.NewGroup("/rating").Tag("rating").
					AddRoutes(
						route.GET("").To(rating).
							Response(Rating{}).
							ShortDesc("Get average rating"),
					),
			),
	}
	tree.AddToWebService(ws)
	return nil
}
