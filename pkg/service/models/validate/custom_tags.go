package validate

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

func registerTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		if err := trans.Add(tag, msg, false); err != nil {
			return err
		}
		return nil
	}
}

func translateFn(trans ut.Translator, fe validator.FieldError) string {
	msg, err := trans.T(fe.Tag(), fe.Field(), fe.Value().(string), fe.Param())
	if err != nil {
		panic(fe.(error).Error())
	}
	return msg
}

func (v *Validator) registCustomTags() error {
	if e := v.Validator.RegisterTranslation("dbuniq", v.Translator, registerTranslator("dbuniq", "{0} 为 {1} 的 {2} 已经存在了"), translateFn); e != nil {
		return e
	}
	if e := v.Validator.RegisterTranslation("reluniq", v.Translator, registerTranslator("reluniq", "该用户已经存在{0}的成员中,请勿重复添加"), translateFn); e != nil {
		return e
	}
	if e := v.Validator.RegisterTranslation("noinchoice", v.Translator, registerTranslator("noinchoice", "{0}错误,非法的选项"), translateFn); e != nil {
		return e
	}
	if e := v.Validator.RegisterTranslation("fqdn", v.Translator, registerTranslator("fqdn", "{0}错误，{1}不是合法的租户名字"), translateFn); e != nil {
		return e
	}
	if e := v.Validator.RegisterTranslation("password", v.Translator, registerTranslator("password", "密码长度至少8位,包含大小写字母和数字以及特殊字符(.!@#$%~)"), translateFn); e != nil {
		return e
	}
	return nil
}
