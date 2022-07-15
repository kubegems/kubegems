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
	"kubegems.io/kubegems/pkg/service/models"
)

func (v *Validator) EnvironmentUserRelStructLevelValidation(sl validator.StructLevel) {
	rel := sl.Current().Interface().(models.EnvironmentUserRels)
	tmp := models.EnvironmentUserRels{}
	if rel.ID == 0 {
		if e := v.db.First(&tmp, "environment_id = ? and user_id = ?", rel.EnvironmentID, rel.UserID).Error; e == nil {
			sl.ReportError(rel.Role, "用户", "Role", "reluniq", "环境")
		}
	}
}
