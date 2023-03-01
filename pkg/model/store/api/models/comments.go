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

package models

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

func GetUsername(req *restful.Request) string {
	username, _ := req.Attribute("username").(string)
	return username
}

type CommentResponse struct {
	repository.Comment
	Replies []repository.Comment `json:"replies"`
}

func (m *ModelsAPI) ListComments(req *restful.Request, resp *restful.Response) {
	postid := postidof(DecodeSourceModelName(req))

	withRepliesCount := request.Query(req.Request, "withRepliesCount", false)
	withReplies := request.Query(req.Request, "withReplies", false)

	listOptions := repository.ListCommentOptions{
		CommonListOptions: ParseCommonListOptions(req),
		PostID:            postid,
		ReplyToID:         req.QueryParameter("reply"),
		WithReplies:       withReplies || withRepliesCount,
	}
	list, err := m.CommentRepository.List(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	total, _ := m.CommentRepository.Count(req.Request.Context(), listOptions)
	response.OK(resp, response.Page[repository.CommentWithAddtional]{
		List:  list,
		Total: total,
		Page:  listOptions.Page,
		Size:  listOptions.Size,
	})
}

func (m *ModelsAPI) CreateComment(req *restful.Request, resp *restful.Response) {
	postid := postidof(DecodeSourceModelName(req))

	comment := &repository.Comment{}
	if err := req.ReadEntity(comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// comment username
	comment.Username = GetUsername(req)
	if err := m.CommentRepository.Create(req.Request.Context(), postid, comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, comment)
}

func (m *ModelsAPI) UpdateComment(req *restful.Request, resp *restful.Response) {
	comment := &repository.Comment{}
	if err := req.ReadEntity(comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	comment.ID = req.PathParameter("comment")
	// comment username
	comment.Username = GetUsername(req)
	if err := m.CommentRepository.Update(req.Request.Context(), comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, comment)
}

func (m *ModelsAPI) DeleteComment(req *restful.Request, resp *restful.Response) {
	// check if user is the owner of the comment
	comment := &repository.Comment{
		ID:       req.PathParameter("comment"),
		Username: GetUsername(req),
	}
	if err := m.CommentRepository.Delete(req.Request.Context(), comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, comment)
}

func (m *ModelsAPI) GetRating(req *restful.Request, resp *restful.Response) {
	postid := postidof(DecodeSourceModelName(req))

	rating, err := m.CommentRepository.Rating(req.Request.Context(), postid)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if len(rating) == 0 {
		// return empty rating
		response.OK(resp, repository.Rating{})
		return
	}
	response.OK(resp, rating[0])
}

func postidof(source, name string) string {
	return source + "/" + name
}

func (m *ModelsAPI) registerCommentsRoute() *route.Group {
	return route.
		NewGroup("/comments").Tag("comments").
		AddRoutes(
			route.GET("").To(m.ListComments).
				Doc("list comments").
				Response([]repository.Comment{}).
				Paged().
				Parameters(
					route.QueryParameter("reply", "reply id,list comments reply to the id").Optional(),
					route.QueryParameter("withReplies", "with replies in list result").Optional().DataType("boolean"),
					route.QueryParameter("withRepliesCount", "with replies count in list result").Optional().DataType("boolean"),
				),
			route.POST("").To(m.CreateComment).Doc("create comment").Parameters(
				route.BodyParameter("comment", repository.Comment{}).Desc(
					"To add a comment,keep field 'replyTo' empty;"+
						"To add a reply comment,set field 'replyTo.rootID' to the comment id;"+
						"To add a reply to reply,set field 'replyTo.rootID' to the top comment id and field 'replyTo.id' to the reply id.",
				),
			),
			route.PUT("/{comment}").To(m.UpdateComment).Doc("update comment").Parameters(
				route.PathParameter("comment", "comment id"),
				route.BodyParameter("comment", repository.Comment{}),
			),
			route.DELETE("/{comment}").To(m.DeleteComment).Doc("delete comment").Parameters(
				route.PathParameter("comment", "comment id"),
			),
		)
}
