package validate

import (
	"github.com/go-playground/validator/v10"
	"github.com/kubegems/gems/pkg/models"
)

func (v *Validator) UserStructLevelValidation(sl validator.StructLevel) {
	user := sl.Current().Interface().(models.User)
	tmp := models.User{}
	// 新创建的时候，用户名不能重名
	if user.ID == 0 && len(user.Username) > 0 {
		if e := v.db.First(&tmp, "username = ?", user.Username).Error; e == nil {
			sl.ReportError(user.Username, "用户名", "Username", "dbuniq", "用户")
		}
	}
	// 修改用户的时候，不能用户名不能重名
	if user.ID != 0 {
		if e := v.db.First(&tmp, "username = ? and id <> ?", user.Username, user.ID).Error; e == nil {
			sl.ReportError(user.Username, "用户名", "Username", "dbuniq", "用户")
		}
	}
}
