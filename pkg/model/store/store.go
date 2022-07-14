package store

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-logr/logr"
	gomongo "go.mongodb.org/mongo-driver/mongo"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/model/store/api/modeldeployments"
	"kubegems.io/kubegems/pkg/model/store/api/models"
	"kubegems.io/kubegems/pkg/model/store/auth"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/jwt"
	"kubegems.io/kubegems/pkg/utils/mongo"
)

type StoreOptions struct {
	Listen string            `json:"listen,omitempty"`
	Mongo  *mongo.Options    `json:"mongo,omitempty"`
	Mysql  *database.Options `json:"mysql,omitempty"`
	Jwt    *jwt.Options      `json:"jwt,omitempty"`
}

func DefaultOptions() *StoreOptions {
	return &StoreOptions{
		Listen: ":8080",
		Mongo:  mongo.DefaultOptions(),
		Mysql:  database.NewDefaultOptions(),
		Jwt:    jwt.DefaultOptions(),
	}
}

func Run(ctx context.Context, options *StoreOptions) error {
	ctx = log.NewContext(ctx, log.LogrLogger)


	server := StoreServer{
		Options: options,
	}
	return server.Run(ctx)
}

type StoreServer struct {
	Options *StoreOptions
}

func (s *StoreServer) Run(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)

	// setup mongodb
	mongocli, mongodb, err := mongo.NewMongoDB(ctx, s.Options.Mongo)
	if err != nil {
		return fmt.Errorf("setup mongo: %v", err)
	}
	defer mongocli.Disconnect(ctx)

	db, err := database.NewDatabase(s.Options.Mysql)
	if err != nil {
		return fmt.Errorf("setup mysql: %v", err)
	}
	agents, err := agents.NewClientSet(db)
	if err != nil {
		return fmt.Errorf("setup agents: %v", err)
	}

	// setup api
	handler, err := s.SetupAPI(ctx, APIDependencies{
		Mongo:    mongodb,
		Authc:    auth.NewJWTAuthenticationManager(s.Options.Jwt),
		Database: db,
		Agents:   agents,
	})
	if err != nil {
		return fmt.Errorf("setup api: %v", err)
	}

	log.Info("start model store service", "listen", s.Options.Listen)
	server := &http.Server{Addr: s.Options.Listen, Handler: handler}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	return server.ListenAndServe()
}

type APIDependencies struct {
	Agents   *agents.ClientSet
	Database *database.Database
	Mongo    *gomongo.Database
	Authc    auth.AuthenticationManager
}

func (s *StoreServer) SetupAPI(ctx context.Context, deps APIDependencies) (http.Handler, error) {
	// models api
	modelsapi, err := models.NewModelsAPI(ctx, deps.Mongo)
	if err != nil {
		return nil, fmt.Errorf("setup models api: %v", err)
	}
	// modeldeployment api
	modeldeploymentsapi := modeldeployments.NewModelDeploymentAPI(deps.Agents, deps.Database)
	return apiutil.NewRestfulAPI("v1",
		[]restful.FilterFunction{
			AuthenticationMiddleware(deps.Authc),
		},
		[]apiutil.RestModule{
			modelsapi,
			modeldeploymentsapi,
		},
	), nil
}

func AuthenticationMiddleware(authc auth.AuthenticationManager) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		whitelist := []string{
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
		info, err := authc.UserInfo(req.Request.Context(), bearerToken)
		if err != nil {
			resp.WriteHeader(http.StatusUnauthorized)
			return
		}
		// set user info to request,get userinfo using:
		//  username, _ := req.Attribute("username").(string)
		req.SetAttribute("username", info.Username)
		chain.ProcessFilter(req, resp)
	}
}
