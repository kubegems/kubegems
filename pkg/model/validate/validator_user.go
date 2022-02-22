package validate

import (
	"context"

	"github.com/go-playground/validator/v10"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
)

func UserStructLevelValidation(sl validator.StructLevel) {
	user := sl.Current().Interface().(forms.UserDetail)
	// 新创建的时候，用户名不能重名
	var c int64
	opts := []client.Option{}
	if user.ID == 0 && len(user.Name) > 0 {
		opts = append(opts, client.Where("username", client.Eq, user.Name))
		modelClient.Count(context.Background(), user.Object(), &c, opts...)
		if c > 0 {
			sl.ReportError("username", "用户名", "Username", "dbuniq", "用户")
		}
	}
	// 修改用户的时候，不能用户名不能重名
	if user.ID != 0 {
		opts = append(opts, client.Where("id", client.Eq, user.ID))
		modelClient.Count(context.Background(), user.Object(), &c, opts...)
		if c > 0 {
			sl.ReportError("username", "用户名", "Username", "dbuniq", "用户")
		}
	}
}
