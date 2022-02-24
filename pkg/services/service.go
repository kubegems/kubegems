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
)

func ServiceContainer(modelClient client.ModelClientIface) *restful.Container {
	servicesContainer := restful.NewContainer()

	BaseHandler := base.NewBaseHandler(nil, modelClient, nil)

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
		applicationhandler.MustNewApplicationDeployHandler(nil, nil, BaseHandler),
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

	sr := ServiceContainer(mc)

	restful.DefaultContainer.RegisteredWebServices()

	http.ListenAndServe(":9090", sr)
}
