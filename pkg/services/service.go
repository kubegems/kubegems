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
	clusterhandler "kubegems.io/pkg/services/handlers/clusters"
	userhandler "kubegems.io/pkg/services/handlers/users"
)

func ServiceContainer(modelClient client.ModelClientIface) *restful.Container {
	servicesContainer := restful.NewContainer()

	regist(
		servicesContainer,
		&userhandler.Handler{
			Path:        "users",
			ModelClient: modelClient,
		},
		&clusterhandler.Handler{
			Path:        "clusters",
			ModelClient: modelClient,
		},
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
	http.ListenAndServe(":9090", sr)
}
