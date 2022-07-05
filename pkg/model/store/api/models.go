package api

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type ModelsAPI struct {
	ModelRepository   *repository.ModelsRepository
	CommentRepository *repository.CommentsRepository
	SourcesRepository *repository.SourcesRepository
}

func NewModelsAPI(db *mongo.Database) *ModelsAPI {
	return &ModelsAPI{
		ModelRepository:   repository.NewModelsRepository(db),
		CommentRepository: repository.NewCommentsRepository(db),
		SourcesRepository: repository.NewSourcesRepository(db),
	}
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
	Rating repository.Rating `json:"rating"`
}

func (m *ModelsAPI) ListModels(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
	}
	list, err := m.ModelRepository.List(ctx, listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// ignore total count error
	total, _ := m.ModelRepository.Count(ctx, listOptions)
	ratinglist := m.fillRating(ctx, list)
	response.OK(resp, response.Page{
		List:  ratinglist,
		Total: total,
		Page:  listOptions.Page,
		Size:  listOptions.Size,
	})
}

func (m *ModelsAPI) fillRating(ctx context.Context, list []repository.Model) []ModelResponse {
	ids := make([]string, 0, len(list))
	for _, model := range list {
		ids = append(ids, model.Source+"/"+model.Name)
	}
	// ignore
	ratings, _ := m.CommentRepository.Rating(ctx, ids...)
	ratingmap := make(map[string]repository.Rating)
	for _, item := range ratings {
		ratingmap[item.ID] = item
	}
	ret := make([]ModelResponse, 0, len(list))
	for _, model := range list {
		ret = append(ret, ModelResponse{
			Model:  model,
			Rating: ratingmap[model.ID],
		})
	}
	return ret
}

func (m *ModelsAPI) GetModel(req *restful.Request, resp *restful.Response) {
	source, name := req.PathParameter("source"), req.PathParameter("model")
	model, err := m.ModelRepository.Get(req.Request.Context(), source, name)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
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
	source, name := req.PathParameter("source"), req.PathParameter("model")
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

func (m *ModelsAPI) ListComments(req *restful.Request, resp *restful.Response) {
	postid := req.PathParameter("source") + "/" + req.PathParameter("model")

	listOptions := repository.ListCommentOptions{
		CommonListOptions: ParseCommonListOptions(req),
		PostID:            postid,
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
	postid := req.PathParameter("source") + "/" + req.PathParameter("model")
	info, _ := req.Attribute("user").(UserInfo)

	comment := &repository.Comment{}
	if err := req.ReadEntity(comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// comment username
	comment.Username = info.Username
	if err := m.CommentRepository.Create(req.Request.Context(), postid, comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, comment)
}

func (m *ModelsAPI) UpdateComment(req *restful.Request, resp *restful.Response) {
	info, _ := req.Attribute("user").(UserInfo)

	comment := &repository.Comment{}
	if err := req.ReadEntity(comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	comment.ID = req.PathParameter("comment")
	// comment username
	comment.Username = info.Username
	if err := m.CommentRepository.Update(req.Request.Context(), comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, comment)
}

func (m *ModelsAPI) DeleteComment(req *restful.Request, resp *restful.Response) {
	info, _ := req.Attribute("user").(UserInfo)

	// check if user is the owner of the comment

	comment := &repository.Comment{
		ID:       req.PathParameter("comment"),
		Username: info.Username,
	}
	if err := m.CommentRepository.Delete(req.Request.Context(), comment); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, comment)
}

func (m *ModelsAPI) GetRating(req *restful.Request, resp *restful.Response) {
	postid := req.PathParameter("source") + "/" + req.PathParameter("model")

	rating, err := m.CommentRepository.Rating(req.Request.Context(), postid)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if len(rating) == 0 {
		response.NotFound(resp, "rating not found")
		return
	}
	response.OK(resp, rating)
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
	ModelCount int64 `json:"modelCount"`
	ImageCount int64 `json:"imageCount"`
}

func (m *ModelsAPI) ListSources(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ListSourceOptions{
		CommonListOptions: ParseCommonListOptions(req),
	}
	list, err := m.SourcesRepository.List(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	total, _ := m.SourcesRepository.Count(req.Request.Context(), listOptions)
	response.OK(resp, response.Page{
		List:  list,
		Total: total,
		Page:  listOptions.Page,
		Size:  listOptions.Size,
	})
}

func (m *ModelsAPI) CreateSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{}
	if err := req.ReadEntity(&source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if err := m.SourcesRepository.Create(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

func (m *ModelsAPI) DeleteSource(req *restful.Request, resp *restful.Response) {
	if err := m.SourcesRepository.Delete(req.Request.Context(), req.PathParameter("source")); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, nil)
}

// nolint: gomnd
func ParseCommonListOptions(r *restful.Request) repository.CommonListOptions {
	opts := repository.CommonListOptions{
		Page:   request.Query(r.Request, "page", int64(1)),
		Size:   request.Query(r.Request, "size", int64(10)),
		Search: request.Query(r.Request, "search", ""),
		Sort:   request.Query(r.Request, "sort", ""),
	}
	if opts.Sort == "" {
		opts.Sort = "-creationTime"
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Size < 1 {
		opts.Size = 10
	}
	return opts
}
