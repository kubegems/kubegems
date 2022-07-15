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

package route

import (
	"log"
	"testing"

	"github.com/emicklei/go-restful/v3"
)

func Samplefunc(req *restful.Request, resp *restful.Response) {
	log.Printf("%s %s", req.Request.Method, req.Request.URL)
}

type SampleLoginData struct {
	Username string
	Password string
}

type SampleAnimal struct {
	Name string
	Age  int
	Zoo  string
}

type SampleResponseData struct {
	Message string
	Data    interface{}
	Error   interface{}
}

func Test_Tree_AddToContainer(t *testing.T) {
	tree := &Tree{
		RouteUpdateFunc: func(r *Route) {
			for i := range r.Responses {
				r.Responses[i].Body = SampleResponseData{
					Data: r.Responses[i].Body,
				}
			}
		},
		Group: NewGroup("/v2").
			AddSubGroup(
				NewGroup("/login").
					AddRoutes(
						POST("/").To(Samplefunc).
							Parameters(
								BodyParameter("user credentials", SampleLoginData{}),
							),
					),
				NewGroup("/zoo").
					AddRoutes(
						GET("/").To(Samplefunc),
					).
					AddSubGroup(
						NewGroup("/{zoo}").
							Parameters(
								PathParameter("zoo", "zoo name"),
							).
							AddRoutes(
								GET("/").To(Samplefunc),
							).
							AddSubGroup(
								NewGroup("/animals").Tag("animals").
									AddRoutes(
										GET("/").To(Samplefunc),
									),
							),
					),
			),
	}

	ws := &restful.WebService{}
	tree.AddToWebService(ws)

	routes := ws.Routes()
	for _, route := range routes {
		log.Printf("registerd: %s %s", route.Method, route.Path)
	}
}
