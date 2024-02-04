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

package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-logr/logr"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
	"kubegems.io/kubegems/pkg/utils/redis"
	"sigs.k8s.io/yaml"
)

const SelfClusterAgentAddress = "https://kubegems-local-agent.kubegems-local:8041"

func createDatabaseIfNotExists(ctx context.Context, opts *database.Options) (exists bool, err error) {
	log := logr.FromContextOrDiscard(ctx)

	cfg := opts.ToDriverConfig()
	dbname := cfg.DBName
	cfg.DBName = ""

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return false, err
	}

	tmpdb := sql.OpenDB(connector)
	defer tmpdb.Close()

	showdb := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '%s'", dbname)
	count := 0
	if err := tmpdb.QueryRowContext(ctx, showdb).Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	sqlStr := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE `%s`;", dbname, opts.Collation)
	log.Info("create database", "sql", sqlStr)
	if _, err := tmpdb.Exec(sqlStr); err != nil {
		return false, err
	}
	return false, nil
}

func MigrateDatabaseAndInitData(ctx context.Context, opts *database.Options, migrate, initData bool, globalvalues string) error {
	log := logr.FromContextOrDiscard(ctx)
	log.WithValues("migrate", migrate, "initData", initData, "globalvalues", globalvalues).Info("migrate database and init data")
	// init database schema
	_, err := createDatabaseIfNotExists(ctx, opts)
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(opts)
	if err != nil {
		return err
	}

	if migrate {
		if err := MigrateModels(db.DB()); err != nil {
			return err
		}
	}
	if err := InitClusterData(ctx, db.DB(), globalvalues); err != nil {
		return err
	}
	if initData {
		if err := InitBaseData(db.DB()); err != nil {
			return err
		}
	}
	return nil
}

func InitClusterData(ctx context.Context, db *gorm.DB, globalvalues string) error {
	if globalvalues != "" {
		values := pluginmanager.GlobalValues{}
		if err := yaml.Unmarshal([]byte(globalvalues), &values); err != nil {
			return fmt.Errorf("unmarshal global values: %v", err)
		}
		cluster := &Cluster{
			ID:               1,
			ClusterName:      values.ClusterName,
			AgentAddr:        SelfClusterAgentAddress,
			APIServer:        "https://kubernetes.default.svc",
			Primary:          true, // is manager cluster
			Runtime:          values.Runtime,
			ImageRepo:        values.ImageRegistry + "/" + values.ImageRepository,
			InstallNamespace: "kubegems-local",
		}
		if e := db.FirstOrCreate(&cluster, cluster.ID).Error; e != nil {
			return e
		}
	}
	admin_tenant := &Tenant{
		ID: 1,
		//  admin is not allowed in gitea as organization name
		TenantName: "default",
		IsActive:   true,
		Remark:     "default tenant",
	}
	if e := db.FirstOrCreate(&admin_tenant, admin_tenant.ID).Error; e != nil {
		return e
	}
	return nil
}

// 初始化系统角色和系统管理员
func InitBaseData(db *gorm.DB) error {
	active := true
	sysadmin := SystemRole{ID: 1, RoleName: "系统管理员", RoleCode: "sysadmin"}
	normal := SystemRole{ID: 2, RoleName: "普通用户", RoleCode: "normal"}
	admin := User{
		ID:           1,
		Username:     "admin",
		IsActive:     &active,
		SystemRoleID: 1,
		// 默认密码 demo!@#admin 生成 htpasswd -bnBC 10 "" 'demo!@#admin'| tr -d ':\n'
		Password: "$2y$10$n3GZNQIB8jTMJS//1DY04eoRC7dQiPVp8MbFP/vPcaNJU96/MmPci",
	}
	if e := db.FirstOrCreate(&sysadmin, sysadmin.ID).Error; e != nil {
		return e
	}
	if e := db.FirstOrCreate(&normal, normal.ID).Error; e != nil {
		return e
	}
	if e := db.FirstOrCreate(&admin, admin.ID).Error; e != nil {
		return e
	}

	tpls, err := getPromqlTpls()
	if err != nil {
		return err
	}
	for i := range tpls {
		if err := db.FirstOrCreate(&tpls[i]).Error; err != nil {
			return err
		}
	}

	dashboardTpls, err := getDashboardTpls()
	if err != nil {
		log.Error(err, "get dashboard templates")
		return err
	}
	for i := range dashboardTpls {
		if err := db.FirstOrCreate(&dashboardTpls[i]).Error; err != nil {
			return err
		}
	}
	if err := db.FirstOrCreate(DefaultChannel).Error; err != nil {
		return err
	}
	return nil
}

func MigrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		// 审计表
		&AuditLog{},
		// 用户表
		&User{}, &UserToken{},
		// 系统角色表
		&SystemRole{},
		// 租户表
		&Tenant{},
		// 租户成员关系表
		&TenantUserRels{},
		// 租户集群资源表
		&TenantResourceQuota{},
		// 项目表
		&Project{},
		// 项目成员关系表
		&ProjectUserRels{},
		// 环境表
		&Environment{},
		// 环境成员关系表
		&EnvironmentUserRels{},
		// 应用表
		&Application{},
		// 集群表
		&Cluster{},
		// 镜像仓库表
		&Registry{},
		// 日志查询历史表
		&LogQueryHistory{},
		// 日志查询快照表
		&LogQuerySnapshot{},
		// workload 资源建议表
		&Workload{},
		// 容器资源建议表
		&Container{},
		// 租户集群资源申请表
		&TenantResourceQuotaApply{},
		// ??
		&EnvironmentResource{},
		// 消息表
		&Message{},
		// 用户消息表
		&UserMessageStatus{},
		// helmChart仓库
		&ChartRepo{},
		// 虚拟空间表
		&VirtualSpace{},
		// 虚拟空间用户表
		&VirtualSpaceUserRels{},
		// 虚拟域名表
		&VirtualDomain{},
		// 告警规则
		&AlertRule{}, &AlertReceiver{},
		// 告警信息表
		&AlertInfo{}, &AlertMessage{},
		// alert channels
		&AlertChannel{},
		// 监控面板表
		&MonitorDashboard{}, &MonitorDashboardTpl{},
		// 登陆源
		&AuthSource{},
		// promql templates
		&PromqlTplScope{}, &PromqlTplResource{}, &PromqlTplRule{},
		// 公告
		&Announcement{},
	)
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func GetErrMessage(err error) error {
	me := &mysql.MySQLError{}
	if !errors.As(err, &me) {
		return err
	}
	return FormatMysqlError(me)
}

func FormatMysqlError(me *mysql.MySQLError) error {
	switch me.Number {
	case mysqlerr.ER_DUP_ENTRY:
		return fmt.Errorf("存在重名对象(code=%v)", me.Number)
	case mysqlerr.ER_DATA_TOO_LONG:
		return fmt.Errorf("数据超长(code=%v)", me.Number)
	case mysqlerr.ER_TRUNCATED_WRONG_VALUE:
		return fmt.Errorf("日期格式错误(code=%v)", me.Number)
	case mysqlerr.ER_NO_REFERENCED_ROW_2:
		return fmt.Errorf("系统错误(外键关联数据出错 code=%v)", me.Number)
	case mysqlerr.ER_ROW_IS_REFERENCED_2:
		return fmt.Errorf("系统错误(外键关联数据错误 code=%v)", me.Number)
	default:
		return fmt.Errorf("系统错误(code=%v, message=%v)!", me.Number, me.Message)
	}
}

func NewPromqlTplMapperFromFile() *templates.PromqlTplMapper {
	bts, err := os.ReadFile("config/promql_tpl.yaml")
	if err != nil {
		return &templates.PromqlTplMapper{Err: err}
	}
	scopes := []*PromqlTplScope{}
	if err := yaml.Unmarshal(bts, &scopes); err != nil {
		return &templates.PromqlTplMapper{Err: err}
	}
	ret := &templates.PromqlTplMapper{M: make(map[string]*templates.PromqlTpl)}
	for _, s := range scopes {
		for _, res := range s.Resources {
			for _, r := range res.Rules {
				ret.M[fmt.Sprintf("%s.%s.%s", s.Name, res.Name, r.Name)] = &templates.PromqlTpl{
					ScopeName:        s.Name,
					ScopeShowName:    s.ShowName,
					ResourceName:     res.Name,
					ResourceShowName: res.ShowName,
					RuleName:         r.Name,
					RuleShowName:     r.ShowName,
					Namespaced:       s.Namespaced,
					Expr:             r.Expr,
					Unit:             r.Unit,
					Labels:           r.Labels,
				}
			}
		}
	}
	return ret
}

func getDashboardTpls() ([]*MonitorDashboardTpl, error) {
	ret := []*MonitorDashboardTpl{}
	if err := filepath.Walk("config/dashboards", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v", path, err)
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}
		bts, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tpl := MonitorDashboardTpl{}
		if err := yaml.Unmarshal(bts, &tpl); err != nil {
			return err
		}
		tplGetter := NewPromqlTplMapperFromFile().FindPromqlTpl
		if err := CheckGraphs(tpl.Graphs, "", tplGetter); err != nil {
			return fmt.Errorf("tpl: %s, %v", tpl.Name, err)
		}
		ret = append(ret, &tpl)
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func getPromqlTpls() ([]*PromqlTplRule, error) {
	bts, err := os.ReadFile("config/promql_tpl.yaml")
	if err != nil {
		return nil, err
	}
	scopes := []PromqlTplScope{}
	if err := yaml.Unmarshal(bts, &scopes); err != nil {
		return nil, err
	}

	ret := []*PromqlTplRule{}
	for _, scope := range scopes {
		for _, res := range scope.Resources {
			for _, rule := range res.Rules {
				rule.Resource = &PromqlTplResource{
					ID:       res.ID,
					ScopeID:  &scope.ID,
					Name:     res.Name,
					ShowName: res.ShowName,
					Scope: &PromqlTplScope{
						ID:         scope.ID,
						Name:       scope.Name,
						ShowName:   scope.ShowName,
						Namespaced: scope.Namespaced,
					},
				}
				ret = append(ret, rule)
			}
		}
	}

	return ret, err
}

const WaitPerid = 5 * time.Second

func WaitDatabaseServer(ctx context.Context, opts *database.Options) error {
	log := logr.FromContextOrDiscard(ctx)
	cfg := opts.ToDriverConfig()
	cfg.DBName = ""
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return err
	}
	sqldb := sql.OpenDB(connector)
	return wait.PollImmediateInfiniteWithContext(ctx, WaitPerid, func(ctx context.Context) (done bool, err error) {
		if err := sqldb.PingContext(ctx); err != nil {
			log.Error(err, "wait database")
			return false, nil
		}
		log.Info("database server ready")
		return true, nil
	})
}

func WaitRedis(ctx context.Context, redisopts redis.Options) error {
	log := logr.FromContextOrDiscard(ctx)

	cli, err := redis.NewClient(&redisopts)
	if err != nil {
		return err
	}

	return wait.PollImmediateInfiniteWithContext(ctx, WaitPerid, func(ctx context.Context) (done bool, err error) {
		if err := cli.Client.Ping(ctx).Err(); err != nil {
			log.Error(err, "wait redis")
			return false, nil
		}
		log.Info("redis ready")
		return true, nil
	})
}
