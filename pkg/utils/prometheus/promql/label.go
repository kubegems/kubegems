package promql

import (
	"fmt"

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

func RemoveDuplicated(ms []LabelMatcher) []LabelMatcher {
	ret := []LabelMatcher{}
	s := set.NewSet[string]()
	for _, m := range ms {
		if !s.Has(m.String()) {
			ret = append(ret, m)
			s.Append(m.String())
		}
	}
	return ret
}
