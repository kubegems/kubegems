package validate

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var fqdn_item_reg = regexp.MustCompile("^[a-z][-a-z0-9]{0,32}$")

var fqdn_item validator.Func = func(fl validator.FieldLevel) bool {
	v := fl.Field().String()
	return fqdn_item_reg.MatchString(v)
}
