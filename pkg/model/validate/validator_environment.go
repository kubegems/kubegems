package validate

import (
	"context"

	"github.com/go-playground/validator/v10"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
)

func EnvironmentUserRelStructLevelValidation(sl validator.StructLevel) {
	rel := sl.Current().Interface().(forms.EnvironmentUserRelCommon)
	if rel.ID == 0 {
		var count int64
		cond := []client.Option{
			client.Where("user_id", client.Eq, rel.UserID),
			client.Where("environment_id", client.Eq, rel.EnvironmentID),
		}
		modelClient.Count(context.Background(), rel.AsObject(), &count, cond...)
		if count > 0 {
			sl.ReportError(rel.Role, "用户", "Role", "reluniq", "环境")
		}
	}
}
