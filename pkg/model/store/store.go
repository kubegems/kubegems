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
	"kubegems.io/kubegems/pkg/model/store/api"
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
			Addr:     "mongo:27017",
			Database: "models",
			Username: "",
			Password: "",
		},
	}
}

func Run(ctx context.Context, options *StoreOptions) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	server := StoreServer{
		Options: options,
		authc:   api.NewJWTAuthenticationManager(),
	}
	return server.Run(ctx)
}

type StoreServer struct {
	Options *StoreOptions
	authc   api.AuthenticationManager
}

func (s *StoreServer) Run(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)

	// setup mongodb

	mongocli, err := s.setupMongo(ctx)
	if err != nil {
		return fmt.Errorf("setup mongo: %v", err)
	}
	defer mongocli.Disconnect(ctx)

	handler, err := s.setupAPI(ctx, mongocli.Database(s.Options.MongoDB.Database))
	if err != nil {
		return fmt.Errorf("setup api: %v", err)
	}

	log.Info("start web service", "listen", s.Options.Listen)
	server := &http.Server{Addr: s.Options.Listen, Handler: handler}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	return server.ListenAndServe()
}

func (s *StoreServer) setupMongo(ctx context.Context) (*mongo.Client, error) {
	mongoopt := &options.ClientOptions{
		Hosts: strings.Split(s.Options.MongoDB.Addr, ","),
	}
	if s.Options.MongoDB.Username != "" && s.Options.MongoDB.Password != "" {
		mongoopt.Auth = &options.Credential{
			Username: s.Options.MongoDB.Username,
			Password: s.Options.MongoDB.Password,
		}
	}
	mongocli, err := mongo.NewClient(mongoopt)
	if err != nil {
		return nil, err
	}
	if err := mongocli.Connect(ctx); err != nil {
		return nil, err
	}
	if err := mongocli.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return mongocli, nil
}

func (s *StoreServer) setupAPI(ctx context.Context, db *mongo.Database) (http.Handler, error) {
	// setup web service
	modelsapi := api.NewModelsAPI(db)
	if err := modelsapi.InitSchemas(ctx); err != nil {
		return nil, fmt.Errorf("init schemas: %v", err)
	}
	rg := route.NewGroup("")
	modelsapi.AddToWebService(rg)

	ws := &restful.WebService{}
	(&route.Tree{RouteUpdateFunc: listWrrapperFunc, Group: rg}).AddToWebService(ws)
	ws.Filter(restful.CrossOriginResourceSharing{AllowedHeaders: []string{"*"}, AllowedMethods: []string{"*"}}.Filter)

	c := restful.NewContainer()
	c.Add(ws)
	// enableSwagger
	c.Add(route.BuildOpenAPIWebService(c.RegisteredWebServices(), "docs.json", completeInfo))
	c.Filter(LogFilter)
	// Add container filter to respond to OPTIONS
	c.Filter(restful.OPTIONSFilter())
	// authentication
	c.Filter(s.AuthenticationMiddleware)
	return c, nil
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

func LogFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	log.Info(req.Request.URL.String(), "method", req.Request.Method, "start", start)
	chain.ProcessFilter(req, resp)
	duration := time.Since(start)
	log.Info(req.Request.URL.String(), "method", req.Request.Method, "code", resp.StatusCode(), "duration", duration.String())
}

func listWrrapperFunc(r *route.Route) {
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
}

func (s *StoreServer) AuthenticationMiddleware(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	whitelist := []string{
		"/api/v1/login",
		"/api/v1/register",
		"/docs.json",
	}
	for _, item := range whitelist {
		if strings.HasPrefix(req.Request.URL.Path, item) {
			chain.ProcessFilter(req, resp)
			return
		}
	}

	// get token from header
	token := req.HeaderParameter("Authorization")
	if token == "" {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}
	// get bearer token
	bearerToken := strings.TrimPrefix(token, "Bearer ")
	info, err := s.authc.UserInfo(req.Request.Context(), bearerToken)
	if err != nil {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}
	// set user info to request,get userinfo using:
	//  info, _ := req.Attribute("user").(api.UserInfo)
	req.SetAttribute("user", info)
	chain.ProcessFilter(req, resp)
}
