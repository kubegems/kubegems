package models

import (
	"context"

	"kubegems.io/kubegems/pkg/model/store/api/models"
	"kubegems.io/kubegems/pkg/utils/mongo"
	"kubegems.io/kubegems/pkg/utils/route"
)

type ModelsAPI struct {
	modelsapi *models.ModelsAPI
}

func NewModelsAPI(ctx context.Context, mongoopt *mongo.Options) (*ModelsAPI, error) {
	_, mongodb, err := mongo.NewMongoDB(ctx, mongoopt)
	if err != nil {
		return nil, err
	}
	modelsapi, err := models.NewModelsAPI(ctx, mongodb)
	if err != nil {
		return nil, err
	}
	return &ModelsAPI{modelsapi: modelsapi}, nil
}

func (m *ModelsAPI) RegisterRoute(rg *route.Group) {
	m.modelsapi.RegisterRoute(rg)
}
