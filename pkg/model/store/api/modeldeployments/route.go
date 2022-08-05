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

package modeldeployments

import (
	gomongo "go.mongodb.org/mongo-driver/mongo"
	modelsv1beta1 "kubegems.io/kubegems/pkg/apis/models/v1beta1"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/route"
)

type ModelDeploymentAPI struct {
	Clientset        *agents.ClientSet
	Database         *database.Database
	SourceRepository *repository.SourcesRepository
	ModelRepository  *repository.ModelsRepository
}

func NewModelDeploymentAPI(clientset *agents.ClientSet, database *database.Database, mondodb *gomongo.Database) *ModelDeploymentAPI {
	return &ModelDeploymentAPI{
		Clientset:        clientset,
		Database:         database,
		SourceRepository: repository.NewSourcesRepository(mondodb),
		ModelRepository:  repository.NewModelsRepository(mondodb),
	}
}

func (o *ModelDeploymentAPI) RegisterRoute(rg *route.Group) {
	rg.AddSubGroup(
		route.NewGroup("/sources/{source}/models/{model}").
			Tag("model deployment").
			Parameters(
				route.PathParameter("source", "source name"),
				route.PathParameter("model", "model name"),
			).
			AddRoutes(
				route.GET("/instances").To(o.ListAllModelDeployments).
					Paged().
					Response([]ModelDeploymentOverview{}).
					Doc("list all model deployments of the model"),
			),
		route.NewGroup("/tenants/{tenant}/projects/{project}/environments/{environment}/modeldeployments").
			Parameters(
				route.PathParameter("tenant", "tenant name"),
				route.PathParameter("project", "project name"),
				route.PathParameter("environment", "environment name"),
			).
			Tag("model deployment").
			AddRoutes(
				route.GET("").To(o.ListModelDeployments).
					Response([]modelsv1beta1.ModelDeployment{}).
					Paged().
					Doc("list model deployments"),
				route.GET("/{name}").To(o.GetModelDeployment).Doc("get model deployment").Response(modelsv1beta1.ModelDeployment{}),
				route.POST("").To(o.CreateModelDeployment).Parameters(
					route.BodyParameter("body", modelsv1beta1.ModelDeployment{}),
				),
				route.PUT("/{name}").To(o.UpdateModelDeployment),
				route.DELETE("/{name}").To(o.DeleteModelDeployment),
				route.PATCH("/{name}").To(o.PatchModelDeployment),
			),
	)
}
