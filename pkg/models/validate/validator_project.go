package validate

import (
	"github.com/go-playground/validator/v10"
	"kubegems.io/pkg/models"
)

func (v *Validator) ProjectStructLevelValidation(sl validator.StructLevel) {
	project := sl.Current().Interface().(models.Project)
	tmp := models.Project{}
	// 新创建的时候，同租户下项目名字不能重名
	if project.ID == 0 && len(project.ProjectName) > 0 {
		if e := v.db.First(&tmp, "project_name = ? and tenant_id = ?", project.ProjectName, project.TenantID).Error; e == nil {
			sl.ReportError(project.ProjectName, "项目名字", "ProjectName", "dbuniq", "项目")
		}
	}
	// 修改项目的时候，同租户下项目名字不能重名
	if project.ID != 0 {
		if e := v.db.First(&tmp, "project_name = ? and tenant_id = ? and id <> ?", project.ProjectName, project.TenantID, project.ID).Error; e == nil {
			sl.ReportError(project.ProjectName, "项目名字", "ProjectName", "dbuniq", "项目")
		}
	}
}
