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

package maps

func LabelChanged(origin, newone map[string]string) bool {
	if len(origin) == 0 {
		return true
	}
	for k, v := range newone {
		tmpv, exist := origin[k]
		if !exist {
			return true
		}
		if tmpv != v {
			return true
		}
	}
	return false
}

func DeleteLabels(origin, todel map[string]string) map[string]string {
	if len(origin) == 0 {
		return origin
	}
	for k := range todel {
		delete(origin, k)
	}
	return origin
}

func GetLabels(origin map[string]string, keys []string) map[string]string {
	ret := map[string]string{}
	for _, key := range keys {
		if v, exist := origin[key]; exist {
			ret[key] = v
		}
	}
	return ret
}
