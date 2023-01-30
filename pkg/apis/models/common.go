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

package models

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	LabelModelNameHash     = GroupName + "/name-hash"
	LabelModelSource       = GroupName + "/source"
	AnnotationEnableProbes = GroupName + "/enable-probes"
)

type Properties map[string]interface{}

func (p Properties) ToRawExtension() *runtime.RawExtension {
	raw, _ := json.Marshal(p)
	return &runtime.RawExtension{Raw: raw}
}
