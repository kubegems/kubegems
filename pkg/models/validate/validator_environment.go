package validate

import (
	"github.com/go-playground/validator/v10"
	"kubegems.io/pkg/models"
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
