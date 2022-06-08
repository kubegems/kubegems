package handlers

import (
	"github.com/go-playground/validator/v10"
	"kubegems.io/kubegems/pkg/v2/model/validate"
)

func ParseError(err error) interface{} {
	switch verr := err.(type) {
	case validator.ValidationErrors:
		ret := map[string]string{}
		for _, fieldErr := range verr {
			ret[fieldErr.Field()] = fieldErr.Translate(validate.GetTranslator())
			return ret
		}
	}

	return err.Error()
}
