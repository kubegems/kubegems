package services

import (
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
)

func enableSwagger(c *restful.Container) {
	config := restfulspec.Config{
		WebServices:                   c.RegisteredWebServices(),
		APIPath:                       "/docs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}

	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"*"},
		CookiesAllowed: true,
		Container:      c,
	}
	c.Filter(cors.Filter)
	c.Add(restfulspec.NewOpenAPIService(config))
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "KubeGems",
			Description: "kubegems restapi",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name:  "kubegems",
					Email: "com@cloudminds.com",
				},
			},
			Version: "1.0.0",
		},
	}
	swo.Schemes = []string{"http", "https"}
	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"jwt": spec.APIKeyAuth("Authorization", "header"),
	}
	swo.Security = []map[string][]string{}
	swo.Security = append(swo.Security, map[string][]string{"jwt": {""}})
}
