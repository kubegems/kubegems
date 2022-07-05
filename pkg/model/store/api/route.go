package api

import (
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/route"
)

func (m *ModelsAPI) AddToWebService(rg *route.Group) {
	rg.AddSubGroup(
		route.NewGroup("/sources").Tag("sources").
			AddRoutes(
				route.GET("").To(m.ListSources).Paged().Doc("List sources").Response([]repository.Source{}),
				route.POST("").To(m.CreateSource).Doc("Create source").Parameters(
					route.BodyParameter("source", repository.Source{}),
				),
				route.GET("/{source}").To(m.GetSource).Doc("Get source").Response(repository.Source{}).Parameters(
					route.PathParameter("source", "Source name"),
				),
				route.DELETE("/{source}").To(m.DeleteSource).Doc("Delete source").Parameters(
					route.PathParameter("source", "Source name"),
				),
			),
		route.
			NewGroup("/sources/{source}").Parameters(route.PathParameter("source", "model source name")).Tag("models").
			AddRoutes(
				route.GET("/selectors").To(m.ListSelectors).Doc("list selectors").Response(repository.Selectors{}),
				route.GET("/models/{model}").To(m.GetModel).Doc("get model").Response(repository.Model{}),
				route.POST("/models").To(m.CreateModel).Doc("create model").Parameters(
					route.BodyParameter("model", repository.Model{}),
				),
				route.GET("/models").To(m.ListModels).Paged().Doc("list models").Response([]repository.Model{}).Parameters(
					route.QueryParameter("framework", "framework name").Optional(),
					route.QueryParameter("license", "license name").Optional(),
					route.QueryParameter("search", "search name").Optional(),
					route.QueryParameter("sort",
						`sort string, eg: "-name,-creationtime", "name,-creationtime"the '-' prefix means descending,otherwise ascending"`,
					).Optional(),
					route.QueryParameter("tags", "filter models contains all tags").Optional(),
				),
				route.DELETE("/models/{model}").To(m.DeleteModel).Doc("delete model"),
			).
			AddSubGroup(
				route.NewGroup("/models/{model}").
					Parameters(route.PathParameter("model", "model name")).Tag("comments").
					AddRoutes(
						route.GET("/comments").To(m.ListComments).
							Doc("list comments").
							Response([]repository.Comment{}).
							Paged().
							Parameters(
								route.QueryParameter("reply", "reply id,list comments reply to the id").Optional(),
							),
						route.POST("/comments").To(m.CreateComment).Doc("create comment").Parameters(
							route.BodyParameter("comment", repository.Comment{}),
						),
						route.PUT("/comments/{comment}").To(m.UpdateComment).Doc("update comment").Parameters(
							route.PathParameter("comment", "comment id"),
							route.BodyParameter("comment", repository.Comment{}),
						),
						route.DELETE("/comments/{comment}").To(m.DeleteComment).Doc("delete comment").Parameters(
							route.PathParameter("comment", "comment id"),
						),
						route.GET("/rating").To(m.GetRating).Doc("get rating").Response(repository.Rating{}),
					),
			),
	)
}
