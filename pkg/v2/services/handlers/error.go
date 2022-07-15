// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
