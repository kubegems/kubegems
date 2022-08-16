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
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/VividCortex/mysqlerr"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"sigs.k8s.io/yaml"
)

func createDatabaseIfNotExists(dsn, dbname string) error {
	tmpdb, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer tmpdb.Close()
	sqlStr := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4;", dbname)
	if _, err := tmpdb.Exec(sqlStr); err != nil {
		return err
	}
	if _, err := tmpdb.Exec(sqlStr); err != nil {
		return err
	}
	return nil
}

func MigrateDatabaseAndInitData(opts *database.Options, initData bool) error {
	// init database schema
	if err := createDatabaseIfNotExists(opts.ToDsnWithOutDB()); err != nil {
		return err
	}

	db, err := database.NewDatabase(opts)
	if err != nil {
		return err
	}

	if err := migrateModels(db.DB()); err != nil {
		return err
	}

	if initData {
		if err := initBaseData(db.DB()); err != nil {
			return err
		}
	}
	return nil
}

// 初始化系统角色和系统管理员
func initBaseData(db *gorm.DB) error {
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
	return nil
}

func migrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		// 审计表
		&AuditLog{},
		// 用户表
		&User{},
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
		// 告警信息表
		&AlertInfo{},
		// 告警消息表
		&AlertMessage{},
		// 监控面板表
		&MonitorDashboard{},
		// 配置
		&OnlineConfig{},
		// 登陆源
		&AuthSource{},
		// promql templates
		&PromqlTplScope{}, &PromqlTplResource{}, &PromqlTplRule{},
	)
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func GetErrMessage(err error) string {
	me := &driver.MySQLError{}
	if !errors.As(err, &me) {
		return err.Error()
	}
	switch me.Number {
	case mysqlerr.ER_DUP_ENTRY:
		return fmt.Sprintf("存在重名对象(code=%v)", me.Number)
	case mysqlerr.ER_DATA_TOO_LONG:
		return fmt.Sprintf("数据超长(code=%v)", me.Number)
	case mysqlerr.ER_TRUNCATED_WRONG_VALUE:
		return fmt.Sprintf("日期格式错误(code=%v)", me.Number)
	case mysqlerr.ER_NO_REFERENCED_ROW_2:
		return fmt.Sprintf("系统错误(外键关联数据出错 code=%v)", me.Number)
	case mysqlerr.ER_ROW_IS_REFERENCED_2:
		return fmt.Sprintf("系统错误(外键关联数据错误 code=%v)", me.Number)
	default:
		return fmt.Sprintf("系统错误(code=%v, message=%v)!", me.Number, me.Message)
	}
}

func GetTplFromFile(scope, resource, rule string) (*prometheus.PromqlTpl, error) {
	bts, err := os.ReadFile("config/promql_tpl.yaml")
	if err != nil {
		return nil, err
	}
	scopes := []*PromqlTplScope{}
	if err := yaml.Unmarshal(bts, &scopes); err != nil {
		return nil, err
	}
	for _, s := range scopes {
		if s.Name == scope {
			for _, res := range s.Resources {
				if res.Name == resource {
					for _, r := range res.Rules {
						if r.Name == rule {
							return &prometheus.PromqlTpl{
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
							}, nil
						}
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("scope: %s, resource %s, rule: %s not found", scope, resource, rule)
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
