package models

import (
	"context"

	"kubegems.io/kubegems/pkg/model/store"
	"kubegems.io/kubegems/pkg/model/store/api"
	"kubegems.io/kubegems/pkg/utils/route"
)

type ModelsAPI struct {
	modelsapi *api.ModelsAPI
}

func NewModelsAPI(ctx context.Context, mongoopt *store.MongoDBOptions) (*ModelsAPI, error) {
	mongocli, err := store.SetupMongo(ctx, mongoopt)
	if err != nil {
		return nil, err
	}
	modelsapi := api.NewModelsAPI(ctx, mongocli.Database(mongoopt.Database))
	if err := modelsapi.InitSchemas(ctx); err != nil {
		return nil, err
	}
	return &ModelsAPI{modelsapi: modelsapi}, nil
}

func (m *ModelsAPI) RegisterRoute(rg *route.Group) {
	m.modelsapi.RegisterRoute(rg)
}
