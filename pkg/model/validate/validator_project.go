package validate

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"kubegems.io/pkg/model/forms"
)

func ProjectStructLevelValidation(sl validator.StructLevel) {
	project := sl.Current().Interface().(forms.ProjectCommon)
	// 新创建的时候，同租户下项目名字不能重名
	// 修改项目的时候，同租户下项目名字不能重名
	fmt.Println(project)
	// TODO:
}
