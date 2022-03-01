package validate

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	chTranslations "github.com/go-playground/validator/v10/translations/zh"
	"gorm.io/gorm"
	"kubegems.io/pkg/service/models"
)

var instance *Validator

func InitValidator(db *gorm.DB) {
	v, err := NewValidator(db)
	if err != nil {
		panic(err)
	}
	instance = v
}

// Deprecated 尽量将依赖内置进入需要依赖的结构体内，避免在业务函数中直接引用外部依赖
func Get() *Validator {
	return instance
}

type Validator struct {
	Validator  *validator.Validate
	Translator ut.Translator
	db         *gorm.DB
}

func NewValidator(db *gorm.DB) (*Validator, error) {
	vali := binding.Validator.Engine().(*validator.Validate)
	zhT := zh.New()
	enT := en.New()
	uni := ut.New(enT, zhT, enT)
	trans, _ := uni.GetTranslator("zh")
	if e := chTranslations.RegisterDefaultTranslations(vali, trans); e != nil {
		return nil, e
	}

	v := &Validator{
		Validator:  vali,
		db:         db,
		Translator: trans,
	}
	if err := v.registCustomTags(); err != nil {
		return nil, err
	}
	v.registStructValidates()
	return v, nil
}

func (v *Validator) registStructValidates() {
	// User struct validate
	v.Validator.RegisterStructValidation(v.UserStructLevelValidation, models.User{})
	v.Validator.RegisterStructValidation(v.TenantStructLevelValidation, models.Tenant{})
	v.Validator.RegisterStructValidation(v.ProjectStructLevelValidation, models.Project{})
	v.Validator.RegisterStructValidation(v.TenantUserRelStructLevelValidation, models.TenantUserRels{})
	v.Validator.RegisterStructValidation(v.EnvironmentUserRelStructLevelValidation, models.EnvironmentUserRels{})
	v.Validator.RegisterStructValidation(v.UserCreateStructLevelValidation, models.UserCreate{})
}
