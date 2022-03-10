package services

import (
	"github.com/go-openapi/spec"
)

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "KubeGems",
			Description: "kubegems rest api",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name:  "kubegems",
					Email: "support@kubegems.io",
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
