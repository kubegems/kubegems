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

package database

import (
	"fmt"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type DatabaseHelper struct {
	DB *gorm.DB
}

func (h *DatabaseHelper) SystemUsers() []uint {
	var ret []uint
	if err := h.DB.Raw("select users.id from users where users.is_active = 1").Scan(&ret).Error; err != nil {
		log.Error(err, "get system users")
	}
	return ret
}

func (h *DatabaseHelper) SystemAdmins() []uint {
	var ret []uint
	if err := h.DB.Raw("select users.id from users left join system_roles on users.system_role_id = system_roles.id where system_roles.role_code = 'sysadmin' and users.is_active = 1").Scan(&ret).Error; err != nil {
		log.Error(err, "get system admins")
	}
	return ret
}

func (h *DatabaseHelper) TenantUsers(tenantIDs ...uint) []uint {
	var ret []uint
	if err := h.DB.Raw("select user_id from tenant_user_rels where tenant_id in ?", tenantIDs).Scan(&ret).Error; err != nil {
		log.Error(err, "get tenant users")
	}
	return ret
}

func (h *DatabaseHelper) TenantAdmins(tenantIDs ...uint) []uint {
	var ret []uint
	if err := h.DB.Raw("select user_id from tenant_user_rels where role = 'admin' and tenant_id in ?", tenantIDs).Scan(&ret).Error; err != nil {
		log.Error(err, "get tenant admins")
	}
	return ret
}

func (h *DatabaseHelper) ProjectUsers(projIDs ...uint) []uint {
	var ret []uint
	if err := h.DB.Raw("select user_id from project_user_rels where project_id in ?", projIDs).Scan(&ret).Error; err != nil {
		log.Error(err, "get project users")
	}
	return ret
}

func (h *DatabaseHelper) ProjectAdmins(projIDs ...uint) []uint {
	var ret []uint
	if err := h.DB.Raw("select user_id from project_user_rels where role = 'admin' and project_id in ?", projIDs).Scan(&ret).Error; err != nil {
		log.Error(err, "get project admins")
	}
	return ret
}

func (h *DatabaseHelper) EnvUsers(envIDs ...uint) []uint {
	var ret []uint
	if err := h.DB.Raw("select user_id from environment_user_rels where environment_id in ?", envIDs).Scan(&ret).Error; err != nil {
		log.Error(err, "get environment users")
	}
	return ret
}

func (h *DatabaseHelper) EnvAdmins(envIDs ...uint) []uint {
	var ret []uint
	if err := h.DB.Raw("select user_id from environment_user_rels where role = 'operator' and environment_id in ?", envIDs).Scan(&ret).Error; err != nil {
		log.Error(err, "get environment admins")
	}
	return ret
}

type AlertPosition struct {
	AlertName string
	Namespace string

	ClusterID   uint
	ClusterName string

	TenantID   uint
	TenantName string

	ProjectID   uint
	ProjectName string

	EnvironmentID   uint
	EnvironmentName string

	From string // 告警来自哪里，monitor/logging
}

func (h *DatabaseHelper) GetAlertPosition(cluster, namespace, name, scope, from string) (AlertPosition, error) {
	ret := AlertPosition{}
	if scope == "" || scope == prometheus.ScopeNormal {
		sql := `select environments.id as environment_id, environments.environment_name, environments.cluster_id, projects.id as project_id, projects.project_name, tenants.id as tenant_id, tenants.tenant_name
		from environments left join clusters on environments.cluster_id = clusters.id
			left join projects on environments.project_id = projects.id
				left join tenants on projects.tenant_id = tenants.id
					where clusters.cluster_name = ? and environments.namespace = ?`
		if err := h.DB.Raw(sql, cluster, namespace).Scan(&ret).Error; err != nil {
			log.Error(err, "get alert position")
			return ret, err
		}

		if ret.ClusterID == 0 || ret.EnvironmentID == 0 || ret.ProjectID == 0 || ret.TenantID == 0 {
			err := fmt.Errorf("can't find such resource by cluster %s namesapce %s", cluster, namespace)
			return ret, err
		}
	} else {
		sql := `select clusters.id as cluster_id from clusters where clusters.cluster_name = ?`
		if err := h.DB.Raw(sql, cluster).Scan(&ret).Error; err != nil {
			log.Error(err, "get alert position")
			return ret, err
		}

		if ret.ClusterID == 0 {
			err := fmt.Errorf("can't find such resource by cluster %s", cluster)
			return ret, err
		}
	}

	ret.ClusterName = cluster
	ret.Namespace = namespace
	ret.AlertName = name

	if from == "" {
		ret.From = prometheus.AlertTypeMonitor
	} else {
		ret.From = from
	}
	return ret, nil
}

type EnvInfo struct {
	ClusterName string
	Namespace   string

	TenantName      string
	ProjectName     string
	EnvironmentName string
}

func (h *DatabaseHelper) ClusterNS2EnvMap() (map[string]EnvInfo, error) {
	envInfos := []EnvInfo{}
	sql := `select clusters.cluster_name, environments.namespace, tenants.tenant_name, projects.project_name, environments.environment_name
		from environments left join clusters on environments.cluster_id = clusters.id
			left join projects on environments.project_id = projects.id
				left join tenants on projects.tenant_id = tenants.id`
	if err := h.DB.Raw(sql).Scan(&envInfos).Error; err != nil {
		return nil, errors.Wrap(err, "list env infos")
	}

	ret := map[string]EnvInfo{}
	for _, v := range envInfos {
		ret[v.ClusterName+"/"+v.Namespace] = v
	}

	return ret, nil
}

func (h *DatabaseHelper) FindPromqlTpl(scope, resource, rule string) (*prometheus.PromqlTpl, error) {
	sql := `select promql_tpl_scopes.name as scope_name,
	promql_tpl_scopes.show_name as scope_show_name,
	promql_tpl_resources.name as resource_name,
	promql_tpl_resources.show_name as resource_show_name,
	promql_tpl_rules.name as rule_name,
	promql_tpl_rules.show_name as rule_show_name,
	namespaced, expr, unit, labels
	from promql_tpl_rules left join promql_tpl_resources on promql_tpl_rules.resource_id = promql_tpl_resources.id
		left join promql_tpl_scopes on promql_tpl_resources.scope_id = promql_tpl_scopes.id
	where promql_tpl_scopes.name = ? and promql_tpl_resources.name = ? and promql_tpl_rules.name = ?`
	ret := prometheus.PromqlTpl{}
	res := h.DB.Raw(sql, scope, resource, rule).Scan(&ret)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errors.New("no promql template found")
	}
	return &ret, nil
}
