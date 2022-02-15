package validate

import (
	"github.com/go-playground/validator/v10"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
)

func UserStructLevelValidation(sl validator.StructLevel) {
	user := sl.Current().Interface().(forms.UserDetail)
	// 新创建的时候，用户名不能重名
	var c int64
	q := &client.Query{
		Where: []*client.Cond{},
	}
	if user.ID == 0 && len(user.Username) > 0 {
		q.Where = append(q.Where, &client.Cond{Field: "username", Op: client.Eq, Value: user.Username})
		modelClient.Count(user.AsObject(), q, &c)
		if c > 0 {
			sl.ReportError("username", "用户名", "Username", "dbuniq", "用户")
		}
	}
	// 修改用户的时候，不能用户名不能重名
	if user.ID != 0 {
		q.Where = append(q.Where, &client.Cond{Field: "id", Op: client.Neq, Value: user.ID})
		modelClient.Count(user.AsObject(), q, &c)
		if c > 0 {
			sl.ReportError("username", "用户名", "Username", "dbuniq", "用户")
		}
	}
}
