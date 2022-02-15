package validate

import (
	"github.com/go-playground/validator/v10"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
)

func EnvironmentUserRelStructLevelValidation(sl validator.StructLevel) {
	rel := sl.Current().Interface().(forms.EnvironmentUserRelCommon)
	if rel.ID == 0 {
		var count int64
		q := &client.Query{
			Where: []*client.Cond{
				{
					Field: "user_id",
					Op:    client.Eq,
					Value: rel.UserID,
				},
				{
					Field: "environment_id",
					Op:    client.Eq,
					Value: rel.EnvironmentID,
				},
			},
		}
		modelClient.Count(rel.AsObject(), q, &count)
		if count > 0 {
			sl.ReportError(rel.Role, "用户", "Role", "reluniq", "环境")
		}
	}
}
