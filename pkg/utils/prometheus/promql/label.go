// Copyright 2023 The kubegems.io Authors
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

package promql

import (
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/pkg/labels"
	"kubegems.io/kubegems/pkg/utils/set"
)

// MatchType is an enum for label matching types.
type MatchType string

// Possible MatchTypes.
const (
	MatchEqual     MatchType = "="
	MatchNotEqual            = "!="
	MatchRegexp              = "=~"
	MatchNotRegexp           = "!~"
)

type LabelMatcher struct {
	Type  MatchType `json:"type"`
	Name  string    `json:"name"`
	Value string    `json:"value"`
}

func (m *LabelMatcher) ToPromqlLabelMatcher() *labels.Matcher {
	ret := labels.Matcher{
		Name:  m.Name,
		Value: m.Value,
	}
	switch m.Type {
	case MatchEqual:
		ret.Type = labels.MatchEqual
	case MatchNotEqual:
		ret.Type = labels.MatchNotEqual
	case MatchRegexp:
		ret.Type = labels.MatchRegexp
	case MatchNotRegexp:
		ret.Type = labels.MatchNotRegexp
	default:
		ret.Type = labels.MatchEqual
	}
	return &ret
}

func (m *LabelMatcher) String() string {
	if m.Type == "" {
		m.Type = MatchEqual
	}
	return fmt.Sprintf("%s%s%q", m.Name, m.Type, m.Value)
}

func CheckAndRemoveDuplicated(ms []LabelMatcher) ([]LabelMatcher, error) {
	ret := []LabelMatcher{}
	s := set.NewSet[string]()
	for _, m := range ms {
		if m.Type != MatchRegexp && m.Type != MatchNotRegexp {
			if strings.Contains(m.Value, "|") {
				return nil, fmt.Errorf("You can only select multiple when using =~ or !=")
			}
		}
		if !s.Has(m.String()) {
			ret = append(ret, m)
			s.Append(m.String())
		}
	}
	return ret, nil
}
