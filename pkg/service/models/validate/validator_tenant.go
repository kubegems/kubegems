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

package validate

import (
	"github.com/go-playground/validator/v10"
	"k8s.io/apimachinery/pkg/util/validation"
	"kubegems.io/kubegems/pkg/service/models"
)

func (v *Validator) TenantStructLevelValidation(sl validator.StructLevel) {
	tenant := sl.Current().Interface().(models.Tenant)
	tmp := models.Tenant{}
	// 租户名字必须符合DNS-1035格式
	if errs := validation.IsDNS1035Label(tenant.TenantName); len(errs) > 0 {
		sl.ReportError(tenant.TenantName, "租户名字", "TenantName", "DNS-1035", "租户")
	}

	// 新创建的时候，用户名不能重名
	if tenant.ID == 0 && len(tenant.TenantName) > 0 {
		var count int64
		if v.db.Find(&tmp, "tenant_name = ?", tenant.TenantName).Count(&count); count != 0 {
			sl.ReportError(tenant.TenantName, "租户名字", "TenantName", "dbuniq", "租户")
		}
	}
	// 修改用户的时候，不能用户名不能重名
	if tenant.ID != 0 {
		var count int64
		if v.db.Find(&tmp, "tenant_name = ? and id <> ?", tenant.TenantName, tenant.ID).Count(&count); count != 0 {
			sl.ReportError(tenant.TenantName, "租户名字", "TenantName", "dbuniq", "租户")
		}
	}
}

func (v *Validator) TenantUserRelStructLevelValidation(sl validator.StructLevel) {
	rel := sl.Current().Interface().(models.TenantUserRels)
	tmp := models.TenantUserRels{}

	// 新创建关系
	if rel.ID == 0 {
		var count int64
		if v.db.Find(&tmp, "user_id = ? and tenant_id = ?", rel.UserID, rel.TenantID).Count(&count); count != 0 {
			sl.ReportError(rel.Role, "租户名字", "", "reluniq", "租户")
		}
	}
	if rel.Role != "admin" && rel.Role != "ordinary" {
		sl.ReportError(rel.Role, "租户角色", "", "noinchoice", "租户")
	}
}
