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
		// sources
		m.registerSourcesRoute(),
		// sources subresources
		route.
			NewGroup("/sources/{source}").Parameters(route.PathParameter("source", "model source name")).
			AddRoutes(
				route.GET("/selectors").To(m.ListSelectors).Tag("sources").Doc("list selectors").Response(repository.Selectors{}),
			).
			AddSubGroup(
				// source admin
				m.registerSourceAdminRoute(),
				// source sync
				m.registerSourceSyncRoute(),
				// models
				m.registerModelsRoute(),
				// models subresources
				route.
					NewGroup("/models/{model}").Parameters(route.PathParameter("model", "model name")).
					AddRoutes(
						// models ratings
						route.GET("/rating").To(m.GetRating).Tag("models").Doc("get rating").Response(repository.Rating{}),
					).
					AddSubGroup(
						// models comments
						m.registerCommentsRoute(),
					),
			),
	)
}
