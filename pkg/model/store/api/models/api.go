package models

import (
	"context"
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"kubegems.io/kubegems/pkg/model/store/auth"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/route"
)

type ModelsAPI struct {
	ModelRepository   *repository.ModelsRepository
	CommentRepository *repository.CommentsRepository
	SourcesRepository *repository.SourcesRepository
	SyncService       *SyncService

	authorization auth.AuthorizationManager
}

func NewModelsAPI(ctx context.Context, db *mongo.Database, syncopt *SyncOptions) (*ModelsAPI, error) {
	api := &ModelsAPI{
		ModelRepository:   repository.NewModelsRepository(db),
		CommentRepository: repository.NewCommentsRepository(db),
		SourcesRepository: repository.NewSourcesRepository(db),
		authorization:     auth.NewLocalAuthorization(ctx, db),
		SyncService:       NewSyncService(syncopt),
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

func (m *ModelsAPI) RegisterRoute(rg *route.Group) {
	rg.AddSubGroup(
		// admin
		m.registerAdminRoute(),
		// sources
		route.
			NewGroup("/sources").Tag("sources").
			AddRoutes(
				route.GET("").To(m.ListSources).Doc("List sources").Response([]repository.Source{}),
				route.GET("/{source}").To(m.GetSource).Doc("Get source").
					Parameters(route.PathParameter("source", "Source name")).
					Response(SourceWithSyncStatus{}),
				route.GET("/{source}/selectors").To(m.ListSelectors).Doc("List selectors").Response([]repository.Selectors{}),
			),
		route.
			NewGroup("/sources/{source}").Parameters(route.PathParameter("source", "model source name")).
			AddSubGroup(
				// source admin
				route.NewGroup("/admins").Tag("admins").AddRoutes(
					route.GET("").To(m.ListSourceAdmin).Doc("list admins").Response([]string{}),
					route.POST("/{username}").To(m.AddSourceAdmin).Doc("add source admin").
						Parameters(route.PathParameter("username", "Username of admin")).Accept("*/*"),
					route.DELETE("/{username}").To(m.DeleteSourceAdmin).Doc("delete source admin").Parameters(
						route.PathParameter("username", "Username of admin"),
					),
				),
				// source models
				route.
					NewGroup("/models").Tag("models").
					AddRoutes(
						route.GET("").To(m.ListModels).Paged().Doc("list models").
							Parameters(
								route.QueryParameter("framework", "framework name").Optional(),
								route.QueryParameter("license", "license name").Optional(),
								route.QueryParameter("search", "search name").Optional(),
								route.QueryParameter("tags", "filter models contains all tags").Optional(),
								route.QueryParameter("task", "task").Optional(),
								route.QueryParameter("framework", "framework").Optional(),
								route.QueryParameter("sort",
									`sort string, eg: "-name,-creationtime", "name,-creationtime"the '-' prefix means descending,otherwise ascending"`,
								).Optional(),
							).
							Response([]repository.Model{}),
						route.GET("/{model}").To(m.GetModel).Doc("get model").
							Parameters(route.PathParameter("model", "model name, base64 encoded name string")).
							Response(repository.Model{}),
						// models ratings
						route.GET("/{model}/rating").To(m.GetRating).Doc("get model rating").
							Parameters(route.PathParameter("model", "model name, base64 encoded name string")).
							Response(repository.Rating{}),
					),
				// models comments
				route.
					NewGroup("/models/{model}").Parameters(route.PathParameter("model", "model name")).
					AddSubGroup(m.registerCommentsRoute()),
			),
	)
}
