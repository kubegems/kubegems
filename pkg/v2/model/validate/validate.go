package validate

import (
	"log"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	chTranslations "github.com/go-playground/validator/v10/translations/zh"
	"kubegems.io/pkg/model/forms"
)

type ValidatorIface interface {
	Validate(value interface{}) map[string]string
}

var (
	v     *validator.Validate
	trans ut.Translator
)

func GetTranslator() ut.Translator {
	if trans == nil {
		panic("validator not initialized")
	}
	return trans
}

func GetValidator() *validator.Validate {
	if v == nil {
		panic("validator not initialized")
	}
	return v
}

type Validator struct {
	V *validator.Validate
	T ut.Translator
}

func (v *Validator) Validate(value interface{}) map[string]string {
	if err := v.V.Struct(value); err != nil {
		errs := err.(validator.ValidationErrors)
		transerr := errs.Translate(v.T)
		return transerr
	}
	return nil
}

func InitValidator() ValidatorIface {
	v = validator.New()
	zhT := zh.New()
	enT := en.New()
	uni := ut.New(enT, zhT, enT)
	_trans, found := uni.GetTranslator("zh")
	if !found {
		log.Fatal("failed to get validator zh trans")
	}
	if e := chTranslations.RegisterDefaultTranslations(v, _trans); e != nil {
		log.Fatal(e)
	}
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	trans = _trans
	registCustomTags()
	registStructValidates()
	return &Validator{V: v, T: _trans}
}

func registStructValidates() {
	// User struct validate
	GetValidator().RegisterStructValidation(TenantStructLevelValidation, forms.TenantCommon{})
	GetValidator().RegisterStructValidation(ProjectStructLevelValidation, forms.ProjectCommon{})
	GetValidator().RegisterStructValidation(TenantUserRelStructLevelValidation, forms.TenantUserRelCommon{})
}
