package services

import (
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/orm"
	"kubegems.io/pkg/model/validate"
	applicationhandler "kubegems.io/pkg/services/handlers/application"
	approvehandler "kubegems.io/pkg/services/handlers/approve"
	appstorehandler "kubegems.io/pkg/services/handlers/appstore"
	"kubegems.io/pkg/services/handlers/base"
	clusterhandler "kubegems.io/pkg/services/handlers/clusters"
	loginhandler "kubegems.io/pkg/services/handlers/login"
	tenanthandler "kubegems.io/pkg/services/handlers/tenants"
	userhandler "kubegems.io/pkg/services/handlers/users"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/redis"
)

func ServiceContainer(modelClient client.ModelClientIface, redisClient *redis.Client, gitopts *git.Options) *restful.Container {
	servicesContainer := restful.NewContainer()

	BaseHandler := base.NewBaseHandler(nil, modelClient, redisClient)

	regist(
		servicesContainer,
		&loginhandler.Handler{
			BaseHandler: BaseHandler,
		},
		&userhandler.Handler{
			BaseHandler: BaseHandler,
		},
		&clusterhandler.Handler{
			BaseHandler: BaseHandler,
		},
		&tenanthandler.Handler{
			BaseHandler: BaseHandler,
		},
		&approvehandler.Handler{
			BaseHandler: BaseHandler,
		},
		&appstorehandler.Handler{
			// TODO:  add extra options
			AppStoreOpt:       nil,
			ChartMuseumClient: nil,
			BaseHandler:       BaseHandler,
		},
		// app handler
		applicationhandler.MustNewApplicationDeployHandler(gitopts, nil, BaseHandler),
	)

	enableSwagger(servicesContainer)
	return servicesContainer
}

type handlerIface interface {
	Regist(c *restful.Container)
}

func regist(c *restful.Container, h ...handlerIface) {
	for idx := range h {
		h[idx].Regist(c)
	}
}

func Run() {
	// TODO
	panic("Not implemented")
}

// LocalDevRun temporary use of local development
func LocalDevRun() {
	gitopts := &git.Options{
		Username: "root",
		Password: "root",
		Addr:     "http://localhost:13000",
	}
	redisClient, err := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err != nil {
		panic(err)

	}
	loger := log.NewDefaultGormZapLogger()
	log.SetLevel("debug")
	db, err := gorm.Open(sqlite.Open("gorm.sqlite3"), &gorm.Config{
		Logger: loger,
	})
	if err != nil {
		panic(err)
	}
	if err := orm.Migrate(db); err != nil {
		panic(err)
	}
	log.Info("start services")
	mc := orm.NewOrmClient(db)

	// required
	validate.InitValidator(mc)

	sr := ServiceContainer(mc, redisClient, gitopts)

	restful.DefaultContainer.RegisteredWebServices()

	http.ListenAndServe(":9090", sr)
}
