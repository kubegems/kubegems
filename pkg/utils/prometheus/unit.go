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

package prometheus

import (
	"fmt"
	"strings"
)

// - name: short
//   explain: 默认
//   # units: [n, u, m, "", K, Mil, Bil, Tri] # 1000进制
// - name: bytes
//   explain: 数据字节量
//   units: [B, KB, MB, GB, TB, PB] # 1024进制
// - name: bytes/sec
//   explain: 数据字节速率
//   units: [B/s, KB/s, MB/s, GB/s, TB/s, PB/s]
// - name: duration
//   explain: 时间
//   units: [ns, us, ms, s, m, h, d, w]
// - name: percent
//   explain: 百分比
//   units: ["0.0-1.0", "0-100"]

var (
	// 单位表
	UnitValueMap = map[string]UnitValue{
		"B":  defaultUnitValue,
		"KB": {Op: "/", Value: "1024"},
		"MB": {Op: "/", Value: "(1024 * 1024)"},
		"GB": {Op: "/", Value: "(1024 * 1024 * 1024)"},
		"TB": {Op: "/", Value: "(1024 * 1024 * 1024 * 1024)"},
		"PB": {Op: "/", Value: "(1024 * 1024 * 1024 * 1024 * 1024)"},

		"B/s":  defaultUnitValue,
		"KB/s": {Op: "/", Value: "1024"},
		"MB/s": {Op: "/", Value: "(1024 * 1024)"},
		"GB/s": {Op: "/", Value: "(1024 * 1024 * 1024)"},
		"TB/s": {Op: "/", Value: "(1024 * 1024 * 1024 * 1024)"},
		"PB/s": {Op: "/", Value: "(1024 * 1024 * 1024 * 1024 * 1024)"},

		"us": {Op: "*", Value: "(1000 * 1000)"},
		"ms": {Op: "*", Value: "1000"},
		"s":  defaultUnitValue,
		"m":  {Op: "/", Value: "60"},
		"h":  {Op: "/", Value: "(60 * 60)"},
		"d":  {Op: "/", Value: "(24 * 60 * 60)"},
		"w":  {Op: "/", Value: "(7 * 24 * 60 * 60)"},

		"0.0-1.0": {Op: "*", Value: "100", Show: "%"},
		"0-100":   {Show: "%"},
	}

	defaultUnitValue = UnitValue{
		Op:    "*",
		Value: "1",
	}
)

type UnitValue struct {
	Show string

	Op    string
	Value string
}

func ParseUnit(unit string) (UnitValue, error) {
	tmp := strings.SplitN(unit, "-", 2)
	switch len(tmp) {
	case 1:
		if tmp[0] == "short" || tmp[0] == "" {
			return UnitValue{}, nil
		} else {
			return UnitValue{}, fmt.Errorf("unit %s not valid", unit)
		}
	case 2:
		if tmp[0] == "custom" {
			return UnitValue{
				Show: tmp[1],
			}, nil
		}

		if ret, ok := UnitValueMap[tmp[1]]; ok {
			if ret.Show == "" {
				ret.Show = tmp[1]
			}
			return ret, nil
		} else {
			return UnitValue{}, fmt.Errorf("unit %s not valid", unit)
		}
	default:
		return UnitValue{}, fmt.Errorf("unit %s not valid", unit)
	}
}
