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

package v1beta1

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
)

type Values struct {
	Raw    []byte                 `json:"-"`
	Object map[string]interface{} `json:"-"`
}

func init() {
	// https://pkg.go.dev/encoding/gob#Register
	gob.Register(map[string]interface{}{})
}

// DeepCopy indicate how to do a deep copy of Values type
func (v *Values) DeepCopy() *Values {
	if v == nil {
		return nil
	}
	out := Values{}
	if v.Raw != nil {
		out.Raw = make([]byte, len(v.Raw))
		copy(out.Raw, v.Raw)
	}
	if v.Object != nil {
		buf := new(bytes.Buffer)
		gob.NewEncoder(buf).Encode(v.Object)
		gob.NewDecoder(buf).Decode(&out.Object)
	}
	return &out
}

func (v Values) FullFill() Values {
	if v.Object == nil && v.Raw != nil {
		v.UnmarshalJSON(v.Raw)
	}
	if v.Raw == nil && v.Object != nil {
		raw, _ := v.MarshalJSON()
		v.Raw = raw
	}
	return v
}

func (v *Values) UnmarshalJSON(in []byte) error {
	if v == nil {
		return errors.New("runtime.RawExtension: UnmarshalJSON on nil pointer")
	}
	if bytes.Equal(in, []byte("null")) {
		return nil
	}
	v.Raw = make([]byte, len(in))
	copy(v.Raw, in)
	val := map[string]interface{}(nil)
	if err := json.Unmarshal(in, &val); err != nil {
		return err
	}
	v.Object = val
	RemoveNulls(v.Object)
	return nil
}

func (re Values) MarshalJSON() ([]byte, error) {
	if re.Raw == nil {
		if re.Object != nil {
			return json.Marshal(re.Object)
		}
		// Value is an 'object' not null
		return []byte("{}"), nil
	}
	return re.Raw, nil
}

// https://github.com/helm/helm/blob/bed1a42a398b30a63a279d68cc7319ceb4618ec3/pkg/chartutil/coalesce.go#L37
// helm CoalesceValues cant handle nested null,like `{a: {b: null}}`, which want to be `{}`
func RemoveNulls(m interface{}) {
	if m, ok := m.(map[string]interface{}); ok {
		for k, v := range m {
			if val, ok := v.(map[string]interface{}); ok {
				RemoveNulls(val)
				if len(val) == 0 {
					delete(m, k)
				}
				continue
			}
			if v == nil {
				delete(m, k)
				continue
			}
		}
	}
}
