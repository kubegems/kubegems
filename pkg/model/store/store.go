package store

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
	"github.com/go-openapi/spec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

type StoreOptions struct {
	Listen  string         `json:"listen,omitempty"`
	MongoDB MongoDBOptions `json:"mongodb,omitempty"`
}

type MongoDBOptions struct {
	Addr     string `json:"addr,omitempty" description:"mongodb address"`
	Database string `json:"database,omitempty" description:"mongodb database"`
	Username string `json:"username,omitempty" description:"mongodb username"`
	Password string `json:"password,omitempty" description:"mongodb password"`
}

func DefaultStoreOptions() *StoreOptions {
	return &StoreOptions{
		Listen: ":8080",
		MongoDB: MongoDBOptions{
			Addr:     "mongidb:27017",
			Database: "kubegems",
			Username: "root",
			Password: "password",
		},
	}
}

func Run(ctx context.Context, options *StoreOptions) error {
	ctx = log.NewContext(ctx, log.LogrLogger)

	server := StoreServer{
		Options: options,
	}
	server.Setup()
	return server.Run(ctx)
}

type StoreServer struct {
	Options    *StoreOptions
	Mongo      *mongo.Client
	WebService *restful.WebService
}

func (s *StoreServer) Setup() error {
	mongocli, err := mongo.NewClient(&options.ClientOptions{
		Hosts: strings.Split(s.Options.MongoDB.Addr, ","),
		Auth: &options.Credential{
			Username: s.Options.MongoDB.Username,
			Password: s.Options.MongoDB.Password,
		},
	})
	if err != nil {
		return err
	}
	s.Mongo = mongocli

	rg := route.NewGroup("")

	mongodb := mongocli.Database(s.Options.MongoDB.Database)
	// comments
	(&repository.Comments{Collection: mongodb.Collection("comments")}).AddToWebservice(rg)
	(&repository.Models{Collection: mongodb.Collection("models")}).AddToWebService(rg)

	tree := &route.Tree{
		// add response wrapper
		RouteUpdateFunc: func(r *route.Route) {
			paged := false
			for _, item := range r.Params {
				if item.Kind == route.ParamKindQuery && item.Name == "page" {
					paged = true
					break
				}
			}
			for i, v := range r.Responses {
				//  if query parameters exist, response as a paged response
				if paged {
					r.Responses[i].Body = response.Response{Data: response.Page{List: v.Body}}
				} else {
					r.Responses[i].Body = response.Response{Data: v.Body}
				}
			}
		},
		Group: rg,
	}

	ws := &restful.WebService{}
	tree.AddToWebService(ws)

	ws.Filter(restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"*"},
	}.Filter)
	s.WebService = ws
	return nil
}

func (s *StoreServer) Run(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)
	if err := s.Mongo.Connect(ctx); err != nil {
		return err
	}
	defer s.Mongo.Disconnect(ctx)

	log.Info("ping mongodb", "addr", s.Options.MongoDB.Addr)
	// test mongo connection
	if err := s.Mongo.Ping(ctx, nil); err != nil {
		return fmt.Errorf("ping mongodb: %v", err)
	}

	log.Info("start web service", "listen", s.Options.Listen)
	// setup web service
	c := restful.NewContainer()
	c.Add(s.WebService)
	postSetup(c)
	server := &http.Server{Addr: s.Options.Listen, Handler: c}

	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	return server.ListenAndServe()
}

func completeInfo(s *spec.Swagger) {
	s.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "KubeGems",
			Description: "kubegems models api",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name:  "kubegems",
					Email: "support@kubegems.io",
				},
			},
			Version: "1.0.0",
		},
	}
	s.Schemes = []string{"http", "https"}
	s.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"jwt": spec.APIKeyAuth("Authorization", "header"),
	}
	s.Security = []map[string][]string{
		{"jwt": {}},
	}
}

func postSetup(c *restful.Container) {
	// enableSwagger
	c.Add(route.BuildOpenAPIWebService(c.RegisteredWebServices(), "docs.json", completeInfo))
	// Add container filter to enable CORS
	// c.Filter(restful.CrossOriginResourceSharing{
	// 	AllowedHeaders: []string{"*"},
	// 	AllowedMethods: []string{"*"},
	// }.Filter)
	// Add container filter logging
	c.Filter(LogFilter)
	// Add container filter to respond to OPTIONS
	c.Filter(restful.OPTIONSFilter())
}

func LogFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	log.Info(req.Request.URL.String(), "method", req.Request.Method, "start", start)
	chain.ProcessFilter(req, resp)
	duration := time.Since(start)
	log.Info(req.Request.URL.String(), "method", req.Request.Method, "code", resp.StatusCode(), "duration", duration.String())
}
