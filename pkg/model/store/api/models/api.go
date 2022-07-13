package models

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/emicklei/go-restful/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"kubegems.io/kubegems/pkg/model/store/auth"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type ModelsAPI struct {
	ModelRepository   *repository.ModelsRepository
	CommentRepository *repository.CommentsRepository
	SourcesRepository *repository.SourcesRepository

	authorization auth.AuthorizationManager
}

func NewModelsAPI(ctx context.Context, db *mongo.Database) (*ModelsAPI, error) {
	api := &ModelsAPI{
		ModelRepository:   repository.NewModelsRepository(db),
		CommentRepository: repository.NewCommentsRepository(db),
		SourcesRepository: repository.NewSourcesRepository(db),
		authorization:     auth.NewLocalAuthorization(ctx, db),
	}
	if err := api.InitSchemas(ctx); err != nil {
		return nil, fmt.Errorf("init schemas: %v", err)
	}
	return api, nil
}

func (m *ModelsAPI) InitSchemas(ctx context.Context) error {
	if err := m.SourcesRepository.InitSchema(ctx); err != nil {
		return err
	}
	if err := m.ModelRepository.InitSchema(ctx); err != nil {
		return err
	}
	if err := m.CommentRepository.InitSchema(ctx); err != nil {
		return err
	}
	return nil
}

type ModelResponse struct {
	repository.Model
	Rating   repository.Rating `json:"rating"`
	Versions []string          `json:"versions"` // not used
}

func (m *ModelsAPI) ListModels(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
		WithRating:        request.Query(req.Request, "withRating", true),
	}
	list, err := m.ModelRepository.List(ctx, listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// ignore total count error
	total, _ := m.ModelRepository.Count(ctx, listOptions)
	response.OK(resp, response.Page{
		List:  list,
		Total: total,
		Page:  listOptions.Page,
		Size:  listOptions.Size,
	})
}

func postidof(source, name string) string {
	return source + "/" + name
}

func (m *ModelsAPI) GetModel(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	model, err := m.ModelRepository.Get(req.Request.Context(), source, name)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}

func DecodeSourceModelName(req *restful.Request) (string, string) {
	source := req.PathParameter("source")
	name := req.PathParameter("model")

	// model name may contains '/' so we b64encode model name at frontend
	if decoded, _ := base64.StdEncoding.DecodeString(name); len(decoded) != 0 {
		name = string(decoded)
	}

	if decodedname, _ := url.PathUnescape(name); decodedname != "" {
		name = decodedname
	}
	return source, name
}

func (m *ModelsAPI) CreateModel(req *restful.Request, resp *restful.Response) {
	model := repository.Model{}
	if err := req.ReadEntity(&model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if err := m.ModelRepository.Create(req.Request.Context(), model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}

func (m *ModelsAPI) DeleteModel(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	if err := m.ModelRepository.Delete(req.Request.Context(), source, name); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, nil)
}

func (m *ModelsAPI) ListSelectors(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
	}
	selectors, err := m.ModelRepository.ListSelectors(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, selectors)
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
	response.OK(resp, response.Page{
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

func (m *ModelsAPI) GetSource(req *restful.Request, resp *restful.Response) {
	source, err := m.SourcesRepository.Get(req.Request.Context(), req.PathParameter("source"))
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

type ResponseSource struct {
	repository.Source
	Count *SourceCount `json:"count,omitempty"`
}

func (m *ModelsAPI) ListSources(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionNone, func(ctx context.Context) (interface{}, error) {
		listOptions := repository.ListSourceOptions{
			CommonListOptions: ParseCommonListOptions(req),
		}
		list, err := m.SourcesRepository.List(req.Request.Context(), listOptions)
		if err != nil {
			return nil, err
		}
		total, _ := m.SourcesRepository.Count(req.Request.Context(), listOptions)
		withCount := request.Query(req.Request, "count", false)
		retlist := make([]ResponseSource, len(list))
		for i, source := range list {
			retlist[i] = ResponseSource{Source: source}
			if withCount {
				count, _ := m.countSource(req.Request.Context(), source.Name)
				retlist[i].Count = &count
			}
		}
		ret := response.Page{
			List:  retlist,
			Total: total,
			Page:  listOptions.Page,
			Size:  listOptions.Size,
		}
		return ret, nil
	})
}

type SourceCount struct {
	ModelsCount int64 `json:"modelsCount"`
	ImagesCount int64 `json:"imagesCount"`
}

func (m *ModelsAPI) countSource(ctx context.Context, source string) (SourceCount, error) {
	counts := SourceCount{}
	modelcount, err := m.ModelRepository.Count(ctx, repository.ModelListOptions{Source: source})
	if err != nil {
		return counts, err
	}
	counts.ModelsCount = modelcount
	return counts, nil
}

func (m *ModelsAPI) CreateSource(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionAdmin, func(ctx context.Context) (interface{}, error) {
		source := &repository.Source{}
		if err := req.ReadEntity(source); err != nil {
			return nil, err
		}
		if err := m.SourcesRepository.Create(ctx, source); err != nil {
			return nil, err
		}
		return source, nil
	})
}

func (m *ModelsAPI) DeleteSource(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionAdmin, func(ctx context.Context) (interface{}, error) {
		source := &repository.Source{
			Name: req.PathParameter("source"),
		}
		if err := m.SourcesRepository.Delete(ctx, source); err != nil {
			return nil, err
		}
		return source, nil
	})
}

func (m *ModelsAPI) UpdateSource(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionAdmin, func(ctx context.Context) (interface{}, error) {
		source := &repository.Source{}
		if err := req.ReadEntity(source); err != nil {
			return nil, err
		}
		source.Name = req.PathParameter("source")
		if err := m.SourcesRepository.Update(ctx, source); err != nil {
			return nil, err
		}
		return source, nil
	})
}

// nolint: gomnd
func ParseCommonListOptions(r *restful.Request) repository.CommonListOptions {
	opts := repository.CommonListOptions{
		Page:   request.Query(r.Request, "page", int64(1)),
		Size:   request.Query(r.Request, "size", int64(10)),
		Search: request.Query(r.Request, "search", ""),
		Sort:   request.Query(r.Request, "sort", ""),
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Size < 1 {
		opts.Size = 10
	}
	return opts
}

func GetUsername(req *restful.Request) string {
	username, _ := req.Attribute("username").(string)
	return username
}
